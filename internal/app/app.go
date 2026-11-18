package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/gnrfan/smsvoid/internal/smpp"
	"github.com/gnrfan/smsvoid/internal/storage"
	"net"
)

type Service struct{ Store *storage.Store }

func (s *Service) Submit(addr net.Addr, m smpp.Message) (string, uint32) {
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		return "", 8
	}
	m.ID = hex.EncodeToString(b)
	if e := s.Store.Save(context.Background(), addr.String(), m); e != nil {
		return "", 8
	}
	return m.ID, 0
}
