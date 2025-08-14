// cmd/api/static.go
package main

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// IMPORTANT: path is relative to THIS file's dir (cmd/api → repo root → frontend)
//
//go:embed all:frontend
var frontendFS embed.FS

func spaHandler() (http.Handler, error) {
	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		return nil, err
	}
	files := http.FS(sub)
	fileServer := http.FileServer(files)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve real files directly (e.g., /assets/app.js, /favicon.ico) or "/" explicitly.
		p := strings.TrimPrefix(r.URL.Path, "/")
		if r.URL.Path == "/" || strings.Contains(path.Base(r.URL.Path), ".") {
			if r.URL.Path == "/" {
				p = "index.html"
			}
			if f, err := sub.Open(p); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback → index.html for client-routed paths (/about, /pricing, etc.)
		b, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(b))
	}), nil
}
