// AppGrabber server: serves the site and builds a per-selection Windows installer
// by appending the chosen app list to a prebuilt stub .exe.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const magic = "APGRBR01"

type app struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Winget string `json:"winget"`
	URL    string `json:"url,omitempty"`
	Args   string `json:"args,omitempty"`
}

type catalog struct {
	Apps []app `json:"apps"`
}

type payload struct {
	Version int   `json:"v"`
	Built   int64 `json:"built"`
	Apps    []app `json:"apps"`
}

var (
	pubDir  = flag.String("public", "public", "static site directory (contains catalog.json)")
	stubExe = flag.String("stub", "dist/appgrabber-stub.exe", "prebuilt Windows installer stub")
	listen  = flag.String("listen", "127.0.0.1:8090", "listen address")

	mu      sync.RWMutex
	byID    map[string]app
	stubBuf []byte
)

func load() error {
	raw, err := os.ReadFile(filepath.Join(*pubDir, "catalog.json"))
	if err != nil {
		return err
	}
	var c catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return err
	}
	m := make(map[string]app, len(c.Apps))
	for _, a := range c.Apps {
		m[a.ID] = a
	}
	stub, err := os.ReadFile(*stubExe)
	if err != nil {
		return err
	}
	mu.Lock()
	byID, stubBuf = m, stub
	mu.Unlock()
	return nil
}

func build(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Apps []string `json:"apps"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil || len(req.Apps) == 0 || len(req.Apps) > 300 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	mu.RLock()
	p := payload{Version: 1, Built: time.Now().Unix()}
	seen := map[string]bool{}
	for _, id := range req.Apps {
		if a, ok := byID[id]; ok && !seen[id] {
			seen[id] = true
			p.Apps = append(p.Apps, a)
		}
	}
	stub := stubBuf
	mu.RUnlock()
	if len(p.Apps) == 0 {
		http.Error(w, "no valid apps", http.StatusBadRequest)
		return
	}
	body, _ := json.Marshal(p)
	var out bytes.Buffer
	out.Grow(len(stub) + len(body) + 16)
	out.Write(stub)
	out.Write(body)
	var n [8]byte
	binary.LittleEndian.PutUint64(n[:], uint64(len(body)))
	out.Write(n[:])
	out.WriteString(magic)

	log.Printf("build: %d apps for %s", len(p.Apps), r.Header.Get("X-Forwarded-For"))
	w.Header().Set("Content-Type", "application/vnd.microsoft.portable-executable")
	w.Header().Set("Content-Disposition", `attachment; filename="AppGrabber-Installer.exe"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(out.Bytes())
}

func main() {
	flag.Parse()
	if err := load(); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/build", build)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.Handle("/", http.FileServer(http.Dir(*pubDir)))
	log.Printf("AppGrabber listening on %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, mux))
}
