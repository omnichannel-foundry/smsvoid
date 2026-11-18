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
	if c.Version {
		fmt.Println("smsvoid (development build)")
		return
	}
	if c.Info {
		fmt.Printf("Application:    smsvoid\n")
		fmt.Printf("Executable:     %s\n", c.Paths.Executable)
		fmt.Printf("Mode:           %s\n", c.Mode)
		fmt.Printf("Configuration:  %s\n", c.ConfigFile)
		fmt.Printf("Config source:  %s\n", map[bool]string{true: "explicit path", false: "conventional path"}[c.ConfigPath != ""])
		fmt.Printf("Data directory: %s\n", c.Paths.DataDir)
		fmt.Printf("State directory:%s\n", c.Paths.StateDir)
		fmt.Printf("Cache directory:%s\n", c.Paths.CacheDir)
		fmt.Printf("Runtime dir:    %s\n", c.Paths.RuntimeDir)
		fmt.Printf("Log directory:  %s\n", c.Paths.LogDir)
		return
	}
	if c.Doctor {
		fmt.Printf("Application:    smsvoid\n")
		fmt.Printf("Executable:     %s\n", c.Paths.Executable)
		fmt.Printf("Mode:           %s\n", c.Mode)
		fmt.Printf("Configuration:  %s\n", c.ConfigFile)
		if _, err := os.Stat(c.ConfigFile); err == nil {
			fmt.Println("Config status:  present")
		} else {
			fmt.Println("Config status:  missing (defaults will be used)")
		}
		for _, d := range []string{c.Paths.DataDir, c.Paths.StateDir, c.Paths.CacheDir, c.Paths.RuntimeDir, c.Paths.LogDir} {
			if info, err := os.Stat(d); err == nil {
				fmt.Printf("Directory OK:   %s (%s)\n", d, info.Mode())
			} else {
				fmt.Printf("Directory missing: %s\n", d)
			}
		}
		return
	}
	if c.Init {
		if err := config.WriteDefaultConfig(c.ConfigFile); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("created %s\n", c.ConfigFile)
		return
	}
	db, e := storage.Open(c.Driver, c.DSN)
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	s := &smpp.Server{Addr: c.SMPPAddr, Handler: &app.Service{Store: db}, AuthUser: c.SMPPUser, AuthPass: c.SMPPPass, FailedSubmits: c.FailDeliveries}
	errch := make(chan error, 2)
	go func() { errch <- s.ListenAndServe() }()
	httpSrv := &http.Server{Addr: c.HTTPAddr, Handler: (&web.Server{Store: db, User: c.HTTPUser, Pass: c.HTTPPass}).Handler()}
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
