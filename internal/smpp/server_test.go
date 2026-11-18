package smpp

import (
	"encoding/binary"
	"net"
	"testing"
)

type testHandler struct {
	got    Message
	status uint32
}

func (h *testHandler) Submit(_ net.Addr, m Message) (string, uint32) {
	h.got = m
	return "id-1", h.status
}

func TestParseSubmit(t *testing.T) {
	b := submitBody("src", "dst", []byte("hi!"))
	m, err := parseSubmit(b, 17)
	if err != nil {
		t.Fatal(err)
	}
	if m.Source != "src" || m.Destination != "dst" || string(m.Body) != "hi!" || m.Sequence != 17 {
		t.Fatalf("unexpected message: %+v", m)
	}
	for _, bad := range [][]byte{nil, {1}, append(append([]byte{}, b[:len(b)-4]...), byte(9), b[len(b)-3], b[len(b)-2], b[len(b)-1])} {
		if _, err := parseSubmit(bad, 1); err == nil {
			t.Errorf("expected error for %v", bad)
		}
	}
}
func TestResponse(t *testing.T) {
	b := response(submitSM, 8, 42, []byte("x"))
	if binary.BigEndian.Uint32(b) != 17 || binary.BigEndian.Uint32(b[4:]) != submitSM|0x80000000 || binary.BigEndian.Uint32(b[8:]) != 8 || binary.BigEndian.Uint32(b[12:]) != 42 || b[16] != 'x' {
		t.Fatalf("bad response %v", b)
	}
}

func TestServerBindSubmitUnbind(t *testing.T) {
	h := &testHandler{}
	s := &Server{Handler: h}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.ln = ln
	s.conns = map[net.Conn]struct{}{}
	go func() {
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			s.mu.Lock()
			s.conns[c] = struct{}{}
			s.mu.Unlock()
			go s.serve(c)
		}
	}()
	defer s.Close()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	write := func(cmd uint32, seq uint32, body []byte) {
		p := make([]byte, 16+len(body))
		binary.BigEndian.PutUint32(p, uint32(len(p)))
		binary.BigEndian.PutUint32(p[4:], cmd)
		binary.BigEndian.PutUint32(p[12:], seq)
		copy(p[16:], body)
		if _, e := c.Write(p); e != nil {
			t.Fatal(e)
		}
	}
	read := func() []byte {
		h := make([]byte, 16)
		if _, e := c.Read(h); e != nil {
			t.Fatal(e)
		}
		b := make([]byte, binary.BigEndian.Uint32(h)-16)
		if len(b) > 0 {
			if _, e := c.Read(b); e != nil {
				t.Fatal(e)
			}
		}
		return append(h, b...)
	}
	write(bindTransceiver, 1, []byte("user\x00"))
	if got := read(); binary.BigEndian.Uint32(got[8:]) != 0 {
		t.Fatalf("bind response %v", got)
	}
	body := submitBody("a", "b", []byte("ok"))
	write(submitSM, 2, body)
	if got := read(); string(got[16:]) != "id-1\x00" {
		t.Fatalf("submit response %v", got)
	}
	if h.got.Source != "a" || h.got.Destination != "b" || string(h.got.Body) != "ok" {
		t.Fatalf("handler got %+v", h.got)
	}
	write(unbind, 3, nil)
	_ = read()
}

func submitBody(source, destination string, payload []byte) []byte {
	b := []byte{0, 0, 0} // service type, source TON/NPI
	b = append(b, source...)
	b = append(b, 0, 0, 0) // source terminator, dest TON/NPI
	b = append(b, destination...)
	b = append(b, 0)
	b = append(b, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(len(payload))) // remaining mandatory fields
	return append(b, payload...)
}
