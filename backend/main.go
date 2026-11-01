package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Listen, Data, Static, Manager, Worker, WorkerConfig, Logs, Metrics, ConfigSocket string
	SecureCookie                                                                     bool
}
type loginAttempt struct {
	at    time.Time
	count int
}
type App struct {
	cfg           Config
	db            *sql.DB
	logger        *log.Logger
	client        *http.Client
	dummyPassword string
	loginMu       sync.Mutex
	attempts      map[string]loginAttempt
	jobsMu        sync.RWMutex
	jobs          []Job
	jobsUpdated   int64
	jobsError     string
	configMu      sync.Mutex
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, code string) {
	respond(w, status, map[string]string{"code": code})
}
func readJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		fail(w, 400, "INVALID_INPUT")
		return false
	}
	if err := d.Decode(new(any)); err != io.EOF {
		fail(w, 400, "INVALID_INPUT")
		return false
	}
	return true
}
func (a *App) internal(w http.ResponseWriter, err error) {
	a.logger.Printf("request failed: %v", err)
	fail(w, 500, "INTERNAL_ERROR")
}
func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func newApp(cfg Config) (*App, error) {
	db, err := openDatabase(filepath.Join(cfg.Data, "mirrorouc.db"))
	if err != nil {
		return nil, err
	}
	dummy, _ := hashPassword(randomToken())
	a := &App{cfg: cfg, db: db, logger: log.Default(), client: &http.Client{Timeout: 10 * time.Second}, dummyPassword: dummy, attempts: map[string]loginAttempt{}, jobs: []Job{}}
	if _, err = a.record("settings", "site"); errors.Is(err, errMissing) {
		if _, err = a.saveRecord("settings", "site", initialSettings(), 0); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err = os.MkdirAll(filepath.Join(cfg.Data, "uploads"), 0755); err != nil {
		db.Close()
		return nil, err
	}
	return a, nil
}
func (a *App) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.db.PingContext(r.Context()); err != nil {
			fail(w, 503, "DATABASE_UNAVAILABLE")
			return
		}
		respond(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/public", a.public)
	mux.HandleFunc("GET /api/v1/monitoring", a.monitoring)
	mux.HandleFunc("GET /api/v1/job-speeds", a.jobSpeeds)
	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.HandleFunc("GET /api/v1/auth/session", a.session)
	mux.HandleFunc("POST /api/v1/auth/logout", a.logout)
	mux.HandleFunc("PUT /api/v1/auth/password", a.password)
	mux.HandleFunc("GET /api/v1/admin/overview", a.overview)
	mux.HandleFunc("GET /api/v1/admin/audit", a.auditList)
	mux.HandleFunc("GET /api/v1/admin/users", a.users)
	mux.HandleFunc("POST /api/v1/admin/users", a.createUser)
	mux.HandleFunc("GET /api/v1/admin/records/{kind}", a.listRecords)
	mux.HandleFunc("PUT /api/v1/admin/records/{kind}/{id}", a.putRecord)
	mux.HandleFunc("DELETE /api/v1/admin/records/{kind}/{id}", a.deleteRecord)
	mux.HandleFunc("POST /api/v1/admin/uploads", a.upload)
	mux.HandleFunc("POST /api/v1/admin/jobs/{name}/{action}", a.command)
	mux.HandleFunc("GET /api/v1/admin/jobs/{name}/logs", a.logs)
	mux.HandleFunc("GET /api/v1/admin/config", a.configView)
	mux.HandleFunc("POST /api/v1/admin/config/preview", a.configPreview)
	mux.HandleFunc("POST /api/v1/admin/config/apply", a.configApply)
	mux.HandleFunc("GET /api/v1/admin/config/versions", a.configVersions)
	mux.HandleFunc("POST /api/v1/admin/config/rollback", a.configRollback)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "NOT_FOUND") })
	mux.HandleFunc("GET /uploads/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !validName(name) || (!strings.HasSuffix(name, ".png") && !strings.HasSuffix(name, ".jpg")) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(a.cfg.Data, "uploads", name))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.Header().Set("Allow", "GET, HEAD")
			fail(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		path := r.URL.Path
		if path == "/" || path == "/status" || path == "/status/" || path == "/admin" || strings.HasPrefix(path, "/admin/") {
			http.ServeFile(w, r, filepath.Join(a.cfg.Static, "index.html"))
			return
		}
		if strings.HasPrefix(path, "/assets/") {
			http.FileServer(http.Dir(a.cfg.Static)).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				a.logger.Printf("panic: %v", v)
				fail(w, 500, "INTERNAL_ERROR")
			}
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
}
func main() {
	var cfg Config
	var initUser string
	flag.StringVar(&cfg.Listen, "listen", env("MIRROROUC_LISTEN", "127.0.0.1:8081"), "HTTP listen address")
	flag.StringVar(&cfg.Data, "data", env("MIRROROUC_DATA", "./data"), "local data directory")
	flag.StringVar(&cfg.Static, "static", env("MIRROROUC_STATIC", "../dist"), "frontend dist directory")
	flag.StringVar(&cfg.Manager, "manager", env("MIRROROUC_MANAGER", "http://127.0.0.1:12345"), "tunasync Manager URL")
	flag.StringVar(&cfg.Worker, "worker", env("MIRROROUC_WORKER", ""), "worker ID")
	flag.StringVar(&cfg.WorkerConfig, "worker-config", env("MIRROROUC_WORKER_CONFIG", ""), "worker TOML configuration")
	flag.StringVar(&cfg.Logs, "logs", env("MIRROROUC_LOGS", ""), "tunasync log root")
	flag.StringVar(&cfg.Metrics, "metrics", env("MIRROROUC_METRICS", ""), "metrics history JSON")
	flag.StringVar(&cfg.ConfigSocket, "config-socket", env("MIRROROUC_CONFIG_SOCKET", ""), "restricted configuration helper Unix socket")
	flag.BoolVar(&cfg.SecureCookie, "secure-cookies", os.Getenv("MIRROROUC_SECURE_COOKIES") == "true", "require HTTPS cookies")
	flag.StringVar(&initUser, "init-admin", "", "create initial administrator using MIRROROUC_ADMIN_PASSWORD (generated if absent)")
	flag.Parse()
	a, err := newApp(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer a.db.Close()
	if initUser != "" {
		password := os.Getenv("MIRROROUC_ADMIN_PASSWORD")
		generated := password == ""
		if generated {
			password = randomToken()
		}
		hash, err := hashPassword(password)
		if err != nil {
			log.Fatal(err)
		}
		if !validName(initUser) {
			log.Fatal("invalid username")
		}
		if _, err = a.db.Exec("INSERT INTO users(username,password,role,created) VALUES(?,?,'admin',?)", initUser, hash, time.Now().UnixMilli()); err != nil {
			log.Fatal(err)
		}
		if generated {
			os.Stdout.WriteString(password + "\n")
		}
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	a.loadCache()
	go a.poll(ctx)
	srv := &http.Server{Addr: cfg.Listen, Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		<-ctx.Done()
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(stop)
	}()
	log.Printf("MirrorOUC API listening on %s", cfg.Listen)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
