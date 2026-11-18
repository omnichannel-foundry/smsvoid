// Package web exposes captured submissions over HTTP.
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gnrfan/smsvoid/internal/storage"
	"html/template"
)

type Server struct {
	Store *storage.Store
	User  string
	Pass  string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		n, err := s.Store.Count(r.Context())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "smsvoid_messages_total %d\n", n)
	})
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", 405)
			return
		}
		f := parseFilter(r)
		m, e := s.Store.ListFiltered(r.Context(), f)
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		f := parseFilter(r)
		m, e := s.Store.ListFiltered(r.Context(), f)
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = template.Must(template.New("page").Parse(`<h1>SmsVoid</h1><p>SMPP message capture</p><table><tr><th>Received</th><th>From</th><th>To</th><th>Message</th></tr>{{range .}}<tr><td>{{.ReceivedAt}}</td><td>{{.Source}}</td><td>{{.Destination}}</td><td><pre>{{.Body}}</pre></td></tr>{{end}}</table>`)).Execute(w, m)
	})
	return s.auth(mux)
}

func parseFilter(r *http.Request) storage.MessageFilter {
	q := r.URL.Query()
	f := storage.MessageFilter{
		Source:      strings.TrimSpace(q.Get("source")),
		Destination: strings.TrimSpace(q.Get("destination")),
		Query:       strings.TrimSpace(q.Get("q")),
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Offset = n
		}
	}
	return f
}

func (s *Server) auth(next http.Handler) http.Handler {
	if s.User == "" || s.Pass == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != s.User || p != s.Pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="SmsVoid"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
