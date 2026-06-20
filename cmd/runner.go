package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v5"

	"mailgraph/internal/collector"
	"mailgraph/internal/config"
	"mailgraph/internal/web"
)

func daemonize(cfg config.Config) error {
	if cfg.PIDFile != "" {
		if err := os.WriteFile(cfg.PIDFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644); err != nil {
			return fmt.Errorf("write pid file: %w", err)
		}
	}
	return nil
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

func runServer(cfg config.Config) error {
	if cfg.Daemon {
		if err := daemonize(cfg); err != nil {
			return err
		}
	}

	go func() {
		colCfg := cfg
		colCfg.Serve = false
		colCfg.Cat = false
		if err := runCollector(colCfg); err != nil {
			log.Printf("collector error: %v", err)
		}
	}()

	e := echo.New()
	srv := web.New(cfg, mailgraphCSS)
	srv.Register(e)

	sc := echo.StartConfig{Address: cfg.ListenAddr}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if cfg.TLSEnabled {
		log.Printf("Mailgraph served at https://localhost%s/", cfg.ListenAddr)
		return sc.StartTLS(ctx, e, cfg.TLSCertFile, cfg.TLSKeyFile)
	}

	log.Printf("Mailgraph served at http://localhost%s/", cfg.ListenAddr)
	return sc.Start(ctx, e)
}

func runCat(cfg config.Config) error {
	c, err := collector.New(cfg)
	if err != nil {
		return err
	}
	if cfg.Verbose {
		log.Printf("processing logfile %s into %s", cfg.LogFile, cfg.RRDDir)
	}
	return c.Run()
}