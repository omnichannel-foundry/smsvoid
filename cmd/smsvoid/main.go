package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gnrfan/smsvoid/internal/app"
	"github.com/gnrfan/smsvoid/internal/config"
	"github.com/gnrfan/smsvoid/internal/smpp"
	"github.com/gnrfan/smsvoid/internal/storage"
	"github.com/gnrfan/smsvoid/internal/web"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	c, e := config.Load(os.Args[1:])
	if e != nil {
		log.Fatal(e)
	}
	db, e := storage.Open(c.Driver, c.DSN)
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	s := &smpp.Server{Addr: c.SMPPAddr, Handler: &app.Service{Store: db}}
	errch := make(chan error, 2)
	go func() { errch <- s.ListenAndServe() }()
	httpSrv := &http.Server{Addr: c.HTTPAddr, Handler: (&web.Server{Store: db}).Handler()}
	go func() { errch <- httpSrv.ListenAndServe() }()
	log.Printf("SMPP listening on %s; HTTP on %s", c.SMPPAddr, c.HTTPAddr)
	select {
	case <-ctx.Done():
	case e := <-errch:
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Fatal(e)
		}
	}
	_ = s.Close()
	_ = httpSrv.Shutdown(context.Background())
	fmt.Println("smsvoid stopped")
}
