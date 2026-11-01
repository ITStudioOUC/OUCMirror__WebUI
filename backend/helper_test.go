package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestConfigurationHelperTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix domain socket integration runs on Linux")
	}
	c := fixture(t)
	c.login("admin")
	root, err := os.MkdirTemp("", "mouc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	socket := filepath.Join(root, "helper.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	c.app.cfg.ConfigSocket = socket
	config, _ := c.app.loadConfig()
	numbers := config.global
	numbers.Concurrent++
	response := c.call("POST", "/api/v1/admin/config/preview", map[string]any{"baseHash": config.hash, "global": numbers}, true)
	status(t, response, 201)
	var draft configDraft
	json.Unmarshal(response.Body.Bytes(), &draft)
	received := make(chan configDraft, 1)
	go func() {
		connection, e := listener.Accept()
		if e != nil {
			return
		}
		defer connection.Close()
		connection.SetDeadline(time.Now().Add(5 * time.Second))
		var value configDraft
		if json.NewDecoder(connection).Decode(&value) == nil {
			received <- value
		}
		json.NewEncoder(connection).Encode(map[string]string{"code": ""})
	}()
	status(t, c.call("POST", "/api/v1/admin/config/apply", map[string]string{"id": draft.ID}, true), 202)
	select {
	case value := <-received:
		if value.ID != draft.ID || value.BaseHash != config.hash || len(value.Files) != 1 {
			t.Fatal("helper did not receive the validated draft")
		}
	case <-time.After(time.Second):
		t.Fatal("helper request missing")
	}
	version, _ := c.app.readVersion(draft.ID)
	if version.Status != "applied" {
		t.Fatal("helper success not recorded")
	}
	// Only the helper writes in this mode; the ordinary web process leaves files untouched.
	after, _ := c.app.loadConfig()
	if after.hash != config.hash {
		t.Fatal("web process bypassed configuration helper")
	}
}
