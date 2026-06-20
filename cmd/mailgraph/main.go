package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v5"

	"github.com/davidullrich/mailgraph/internal/collector"
	"github.com/davidullrich/mailgraph/internal/config"
	"github.com/davidullrich/mailgraph/internal/web"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mailgraph: %v\n", err)
		os.Exit(1)
	}

	if cfg.Daemon {
		if err := daemonize(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "mailgraph: %v\n", err)
			os.Exit(1)
		}
	}

	if cfg.Cat {
		c, err := collector.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		if cfg.Verbose {
			log.Printf("processing logfile %s into %s", cfg.LogFile, cfg.RRDDir)
		}
		if err := c.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if cfg.Serve {
		go func() {
			if err := runCollector(cfg); err != nil {
				log.Printf("collector error: %v", err)
			}
		}()

		e := echo.New()
		srv := web.New(cfg)
		srv.Register(e)

		log.Printf("Mailgraph served at http://localhost%s/mailgraph/", cfg.ListenAddr)
		if err := e.Start(cfg.ListenAddr); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := runCollector(cfg); err != nil {
		log.Fatal(err)
	}
}

func runCollector(cfg config.Config) error {
	c, err := collector.New(cfg)
	if err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		os.Exit(0)
	}()

	return c.Run()
}

func daemonize(cfg config.Config) error {
	if cfg.PIDFile != "" {
		if err := os.WriteFile(cfg.PIDFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644); err != nil {
			return fmt.Errorf("write pid file: %w", err)
		}
	}
	return nil
}