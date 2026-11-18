// Package smpp implements the SMPP 3.4 server adapter.
package smpp

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bindReceiver    uint32 = 1
	bindTransmitter uint32 = 2
	submitSM        uint32 = 4
	unbind          uint32 = 6
	bindTransceiver uint32 = 9
	enquireLink     uint32 = 0x15
	maxPDU                 = 1 << 20
)

// Message is an accepted mobile terminated message.
type Message struct {
	ID, Source, Destination string
	Body                    []byte
	DataCoding              byte
	RegisteredDelivery      byte
	Sequence                uint32
}

// Handler owns application behavior for submissions.
type Handler interface {
	Submit(net.Addr, Message) (string, uint32)
}

// Server accepts SMPP client connections.
type Server struct {
	Addr          string
	Handler       Handler
	AuthUser      string
	AuthPass      string
	ln            net.Listener
	mu            sync.Mutex
	conns         map[net.Conn]struct{}
	next          atomic.Uint64
	FailedSubmits bool
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	s.mu.Lock()
	s.conns = make(map[net.Conn]struct{})
	s.mu.Unlock()
	for {
		c, e := ln.Accept()
		if e != nil {
			if s.ln == nil {
				return nil
			}
			return e
		}
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		go s.serve(c)
	}
}
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if s.ln != nil {
		err = s.ln.Close()
		s.ln = nil
	}
	for c := range s.conns {
		_ = c.Close()
	}
	return err
}
func (s *Server) serve(c net.Conn) {
	defer func() { _ = c.Close(); s.mu.Lock(); delete(s.conns, c); s.mu.Unlock() }()
	bound := false
	for {
		h := make([]byte, 16)
		if _, e := io.ReadFull(c, h); e != nil {
			return
		}
		n := binary.BigEndian.Uint32(h)
		id := binary.BigEndian.Uint32(h[4:])
		seq := binary.BigEndian.Uint32(h[12:])
		if n < 16 || n > maxPDU {
			return
		}
		b := make([]byte, int(n)-16)
		if _, e := io.ReadFull(c, b); e != nil {
			return
		}
		var out []byte
		switch id {
		case bindReceiver, bindTransmitter, bindTransceiver:
			if bound {
				out = response(id, 5, seq, nil)
			} else {
				systemID, password := "", ""
				i := 0
				if v, e := cstring(b, &i); e == nil {
					systemID = v
				}
				if v, e := cstring(b, &i); e == nil {
					password = v
				}
				if s.AuthUser != "" || s.AuthPass != "" {
					if systemID != s.AuthUser || password != s.AuthPass {
						out = response(id, 14, seq, nil)
						if _, e := c.Write(out); e != nil {
							return
						}
						continue
					}
				}
				bound = true
				out = response(id, 0, seq, []byte("smsvoid\x00"))
			}
		case submitSM:
			if !bound {
				out = response(id, 4, seq, nil)
			} else {
				m, e := parseSubmit(b, seq)
				status := uint32(0)
				mid := ""
				if e != nil {
					status = 8
				} else if s.Handler != nil {
					mid, status = s.Handler.Submit(c.RemoteAddr(), m)
				}
				if e == nil && status == 0 && m.RegisteredDelivery != 0 {
					go func(msg Message, msgID string) {
						time.Sleep(2 * time.Second)
						_, _ = c.Write(receiptPDU(msg.Destination, msg.Source, msgID, msg.Sequence, s.FailedSubmits))
					}(m, mid)
				}
				out = response(id, status, seq, append([]byte(mid), 0))
			}
		case enquireLink:
			out = response(id, 0, seq, nil)
		case unbind:
			out = response(id, 0, seq, nil)
			_, _ = c.Write(out)
			return
		default:
			out = response(0x80000000, 3, seq, nil)
		}
		if _, e := c.Write(out); e != nil {
			return
		}
	}
}
func response(command, status, seq uint32, body []byte) []byte {
	out := make([]byte, 16+len(body))
	binary.BigEndian.PutUint32(out, uint32(len(out)))
	binary.BigEndian.PutUint32(out[4:], command|0x80000000)
	binary.BigEndian.PutUint32(out[8:], status)
	binary.BigEndian.PutUint32(out[12:], seq)
	copy(out[16:], body)
	return out
}
func cstring(b []byte, i *int) (string, error) {
	if *i >= len(b) {
		return "", io.ErrUnexpectedEOF
	}
	j := *i
	for j < len(b) && b[j] != 0 {
		j++
	}
	if j == len(b) {
		return "", io.ErrUnexpectedEOF
	}
	v := string(b[*i:j])
	*i = j + 1
	return v, nil
}
func parseSubmit(b []byte, seq uint32) (Message, error) {
	m := Message{Sequence: seq}
	i := 0
	if _, e := cstring(b, &i); e != nil {
		return m, e
	}
	if i+2 > len(b) {
		return m, io.ErrUnexpectedEOF
	}
	i += 2
	var e error
	if m.Source, e = cstring(b, &i); e != nil {
		return m, e
	}
	if i+2 > len(b) {
		return m, io.ErrUnexpectedEOF
	}
	i += 2
	if m.Destination, e = cstring(b, &i); e != nil {
		return m, e
	}
	if i+3 > len(b) {
		return m, io.ErrUnexpectedEOF
	}
	i += 3 // esm_class, protocol_id, priority_flag
	if _, e = cstring(b, &i); e != nil {
		return m, e
	} // schedule_delivery_time
	if _, e = cstring(b, &i); e != nil {
		return m, e
	} // validity_period
	if i+3 > len(b) {
		return m, io.ErrUnexpectedEOF
	}
	m.RegisteredDelivery = b[i]
	i += 2 // registered_delivery, replace_if_present
	m.DataCoding = b[i]
	i++
	i++
	if i >= len(b) {
		return m, io.ErrUnexpectedEOF
	}
	n := int(b[i])
	i++
	if i+n > len(b) {
		return m, fmt.Errorf("invalid short_message length")
	}
	m.Body = append([]byte(nil), b[i:i+n]...)
	i += n
	// Optional TLVs (including message_payload, tag 0x0424).
	for i+4 <= len(b) {
		tag := binary.BigEndian.Uint16(b[i:])
		length := int(binary.BigEndian.Uint16(b[i+2:]))
		i += 4
		if length < 0 || i+length > len(b) {
			break
		}
		if tag == 0x0424 {
			m.Body = append([]byte(nil), b[i:i+length]...)
		}
		i += length
	}
	return m, nil
}

func receiptPDU(source, destination, id string, seq uint32, failed bool) []byte {
	stamp := time.Now().UTC().Format("0601021504")
	state, status, delivered, stateByte := "DELIVRD", "00000000", "001", byte(2)
	if failed {
		state, status, delivered, stateByte = "UNDELIV", "00000069", "000", byte(5)
	}
	body := fmt.Sprintf("id:%s sub:001 dlvrd:%s submit date:%s done date:%s stat:%s err:%s Text:...", id, delivered, stamp, stamp, state, status)
	p := make([]byte, 16)
	binary.BigEndian.PutUint32(p[4:], 5)
	binary.BigEndian.PutUint32(p[12:], seq)
	p = append(p, 0, 0, 0)
	p = append(p, source...)
	p = append(p, 0, 0, 0)
	p = append(p, destination...)
	p = append(p, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, byte(len(body)))
	p = append(p, body...)
	p = append(p, 0x00, 0x1e, 0x00, byte(len(id)+1))
	p = append(p, id...)
	p = append(p, 0)
	p = append(p, 0x04, 0x27, 0, 1, stateByte)
	binary.BigEndian.PutUint32(p, uint32(len(p)))
	return p
}
