package storage

import (
	"context"
	"github.com/gnrfan/smsvoid/internal/smpp"
	"path/filepath"
	"testing"
)

func TestSQLiteRoundTrip(t *testing.T) {
	s, e := Open("sqlite", filepath.Join(t.TempDir(), "messages.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	m := smpp.Message{ID: "msg-1", Source: "100", Destination: "200", Body: []byte("hello"), DataCoding: 0, Sequence: 7}
	if e = s.Save(context.Background(), "127.0.0.1:1", m); e != nil {
		t.Fatal(e)
	}
	got, e := s.List(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if len(got) != 1 || got[0].ID != m.ID || got[0].Source != m.Source || got[0].Destination != m.Destination || got[0].Body != "hello" || got[0].Sequence != 7 {
		t.Fatalf("unexpected rows: %+v", got)
	}
}
