// Package web exposes captured submissions over HTTP.
package web

import (
	"encoding/json"
	"github.com/gnrfan/smsvoid/internal/storage"
	"html/template"
	"net/http"
)

type Server struct{ Store *storage.Store }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", 405)
			return
		}
		m, e := s.Store.List(r.Context())
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m, e := s.Store.List(r.Context())
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = template.Must(template.New("page").Parse(`<h1>SmsVoid</h1><p>SMPP message capture</p><table><tr><th>Received</th><th>From</th><th>To</th><th>Message</th></tr>{{range .}}<tr><td>{{.ReceivedAt}}</td><td>{{.Source}}</td><td>{{.Destination}}</td><td><pre>{{.Body}}</pre></td></tr>{{end}}</table>`)).Execute(w, m)
	})
	return mux
}
