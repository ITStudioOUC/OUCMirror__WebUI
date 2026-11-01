package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func tokenHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func hashPassword(password string) (string, error) {
	if len(password) < 12 || len(password) > 256 {
		return "", errors.New("INVALID_PASSWORD")
	}
	salt := make([]byte, 24)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	if err != nil {
		return "", err
	}
	return "pbkdf2$600000$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}
func verifyPassword(password, encoded string) bool {
	p := strings.Split(encoded, "$")
	if len(p) != 4 || p[0] != "pbkdf2" || len(password) > 256 {
		return false
	}
	iterations, err := strconv.Atoi(p[1])
	if err != nil || iterations < 100000 || iterations > 1000000 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(p[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(p[3])
	if err != nil || len(want) != 32 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, 32)
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}
func (a *App) sameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		return err == nil && (u.Scheme == "http" || u.Scheme == "https") && strings.EqualFold(u.Host, r.Host)
	}
	return true
}
func (a *App) currentSession(r *http.Request) (Session, error) {
	var s Session
	c, err := r.Cookie("mirrorouc_session")
	if err != nil {
		return s, err
	}
	err = a.db.QueryRow("SELECT users.id,username,role,csrf FROM sessions JOIN users ON users.id=sessions.user_id WHERE token=? AND expires>?", tokenHash(c.Value), time.Now().Unix()).Scan(&s.UserID, &s.Username, &s.Role, &s.CSRF)
	return s, err
}
func (a *App) require(w http.ResponseWriter, r *http.Request, admin bool) (Session, bool) {
	s, err := a.currentSession(r)
	if err != nil {
		fail(w, 401, "UNAUTHORIZED")
		return s, false
	}
	if admin && s.Role != "admin" {
		fail(w, 403, "FORBIDDEN")
		return s, false
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if !a.sameOrigin(r) || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(s.CSRF)) != 1 {
			fail(w, 403, "CSRF_INVALID")
			return s, false
		}
	}
	return s, true
}
func (a *App) session(w http.ResponseWriter, r *http.Request) {
	s, err := a.currentSession(r)
	if err != nil {
		fail(w, 401, "UNAUTHORIZED")
		return
	}
	respond(w, 200, map[string]any{"username": s.Username, "role": s.Role, "csrf": s.CSRF})
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !a.sameOrigin(r) {
		fail(w, 403, "CSRF_INVALID")
		return
	}
	address, _, _ := net.SplitHostPort(r.RemoteAddr)
	a.loginMu.Lock()
	attempt := a.attempts[address]
	if time.Since(attempt.at) > time.Minute {
		attempt = loginAttempt{at: time.Now()}
	}
	attempt.count++
	a.attempts[address] = attempt
	for key, value := range a.attempts {
		if time.Since(value.at) > 10*time.Minute {
			delete(a.attempts, key)
		}
	}
	a.loginMu.Unlock()
	if attempt.count > 10 {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "RATE_LIMITED")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	var s Session
	var encoded string
	err := a.db.QueryRow("SELECT id,username,role,password FROM users WHERE username=?", body.Username).Scan(&s.UserID, &s.Username, &s.Role, &encoded)
	if err != nil {
		encoded = a.dummyPassword
	}
	valid := verifyPassword(body.Password, encoded)
	if err != nil || !valid {
		a.audit(Session{Username: body.Username}, "login", "", "denied")
		fail(w, 401, "INVALID_CREDENTIALS")
		return
	}
	token, csrf := randomToken(), randomToken()
	expires := time.Now().Add(12 * time.Hour)
	if _, err = a.db.Exec("INSERT INTO sessions(token,user_id,csrf,expires) VALUES(?,?,?,?)", tokenHash(token), s.UserID, csrf, expires.Unix()); err != nil {
		a.internal(w, err)
		return
	}
	a.db.Exec("DELETE FROM sessions WHERE expires<?", time.Now().Unix())
	http.SetCookie(w, &http.Cookie{Name: "mirrorouc_session", Value: token, Path: "/", HttpOnly: true, Secure: a.cfg.SecureCookie, SameSite: http.SameSiteStrictMode, MaxAge: 43200, Expires: expires})
	a.audit(s, "login", "", "success")
	respond(w, 200, map[string]any{"username": s.Username, "role": s.Role, "csrf": csrf})
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, false)
	if !ok {
		return
	}
	if c, err := r.Cookie("mirrorouc_session"); err == nil {
		a.db.Exec("DELETE FROM sessions WHERE token=?", tokenHash(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "mirrorouc_session", Value: "", Path: "/", HttpOnly: true, Secure: a.cfg.SecureCookie, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	a.audit(s, "logout", "", "success")
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) password(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, false)
	if !ok {
		return
	}
	var body struct {
		Current  string `json:"current"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	var old string
	if err := a.db.QueryRow("SELECT password FROM users WHERE id=?", s.UserID).Scan(&old); err != nil {
		a.internal(w, err)
		return
	}
	if !verifyPassword(body.Current, old) {
		fail(w, 400, "INVALID_CREDENTIALS")
		return
	}
	hash, err := hashPassword(body.Password)
	if err != nil {
		fail(w, 400, "INVALID_PASSWORD")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		a.internal(w, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec("UPDATE users SET password=? WHERE id=?", hash, s.UserID); err == nil {
		_, err = tx.Exec("DELETE FROM sessions WHERE user_id=?", s.UserID)
	}
	if err != nil {
		a.internal(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		a.internal(w, err)
		return
	}
	a.audit(s, "password", s.Username, "success")
	respond(w, 200, map[string]bool{"ok": true})
}
