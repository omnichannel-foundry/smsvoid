// Package storage provides SQL persistence for captured SMPP submissions.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gnrfan/smsvoid/internal/smpp"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Store struct {
	DB     *sql.DB
	driver string
}

func Open(driver, dsn string) (*Store, error) {
	d := driver
	if d == "postgres" {
		d = "pgx"
	} else if d == "sqlite" {
		d = "sqlite3"
	}
	db, e := sql.Open(d, dsn)
	if e != nil {
		return nil, e
	}
	s := &Store{db, driver}
	q := `CREATE TABLE IF NOT EXISTS messages(id TEXT PRIMARY KEY, received_at TIMESTAMP NOT NULL, remote_addr TEXT NOT NULL, source TEXT NOT NULL, destination TEXT NOT NULL, body BYTEA NOT NULL, data_coding INTEGER NOT NULL, sequence INTEGER NOT NULL)`
	if driver == "sqlite" {
		q = `CREATE TABLE IF NOT EXISTS messages(id TEXT PRIMARY KEY, received_at TEXT NOT NULL, remote_addr TEXT NOT NULL, source TEXT NOT NULL, destination TEXT NOT NULL, body BLOB NOT NULL, data_coding INTEGER NOT NULL, sequence INTEGER NOT NULL)`
	}
	if _, e = db.Exec(q); e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func (s *Store) Save(ctx context.Context, addr string, m smpp.Message) error {
	q := `INSERT INTO messages(id,received_at,remote_addr,source,destination,body,data_coding,sequence) VALUES(?,?,?,?,?,?,?,?)`
	if s.driver == "postgres" {
		q = `INSERT INTO messages(id,received_at,remote_addr,source,destination,body,data_coding,sequence) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`
	}
	_, e := s.DB.ExecContext(ctx, q, m.ID, time.Now().UTC(), addr, m.Source, m.Destination, m.Body, m.DataCoding, m.Sequence)
	return e
}

type Message struct {
	ID, ReceivedAt, RemoteAddr, Source, Destination, Body string
	DataCoding, Sequence                                  int
}

func (s *Store) List(ctx context.Context) ([]Message, error) {
	rows, e := s.DB.QueryContext(ctx, `SELECT id,received_at,remote_addr,source,destination,body,data_coding,sequence FROM messages ORDER BY received_at DESC LIMIT 250`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		var body []byte
		var receivedAt interface{}
		if e = rows.Scan(&m.ID, &receivedAt, &m.RemoteAddr, &m.Source, &m.Destination, &body, &m.DataCoding, &m.Sequence); e != nil {
			return nil, e
		}
		if t, ok := receivedAt.(time.Time); ok {
			m.ReceivedAt = t.UTC().Format(time.RFC3339Nano)
		} else {
			m.ReceivedAt = fmt.Sprint(receivedAt)
		}
		m.Body = string(body)
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) Close() error {
	if s.DB == nil {
		return fmt.Errorf("nil database")
	}
	return s.DB.Close()
}
