package web

import (
	"github.com/gnrfan/smsvoid/internal/storage"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndpoints(t *testing.T) {
	db, e := storage.Open("sqlite", filepath.Join(t.TempDir(), "db.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	h := (&Server{Store: db}).Handler()
	for _, tc := range []struct{ path, content string }{{"/api/v1/messages", "[]"}, {"/", "SmsVoid"}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), tc.content) {
			t.Errorf("%s: status=%d body=%s", tc.path, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/api/v1/messages", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 405 {
		t.Fatalf("POST status %d", w.Code)
	}
}
