package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/prakashniraula/portfolio-in-go/internal/config"
	"github.com/prakashniraula/portfolio-in-go/internal/db"
	httpx "github.com/prakashniraula/portfolio-in-go/internal/http"
	"github.com/prakashniraula/portfolio-in-go/internal/repo"
	"github.com/prakashniraula/portfolio-in-go/internal/service"
	"github.com/prakashniraula/portfolio-in-go/internal/web"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	sqlDB, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(sqlDB, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	store := repo.NewSQLiteStore(sqlDB)
	svc := service.New(store, cfg)

	if cfg.SeedOnEmpty {
		if err := db.SeedIfEmpty(context.Background(), store, cfg.FixturesPath, cfg.AdminUser, cfg.AdminPassword); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	renderer, err := web.NewRenderer(cfg.TemplatesDir, cfg.BaseURL, svc)
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	router := httpx.NewRouter(cfg, svc, renderer)
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatalf("bind %s: %v (is another server still running? try: ss -tlnp | grep %s)", cfg.Addr, err, cfg.Addr)
	}
	log.Printf("listening on %s (db=%s)", ln.Addr().String(), cfg.DBPath)

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
