package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/db"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
)

func main() {
	cfg, err := config.LoadLoginConfig("conf/loginserver/server.properties", "conf/server.properties")
	if err != nil {
		log.Printf("using default login config: %v", err)
		cfg = config.DefaultLoginConfig()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var accounts loginserver.AccountStore = loginserver.NewMemoryAccountStore()
	var games loginserver.GameServerStore = loginserver.NewMemoryGameServerStore()
	if pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 45*time.Second); err == nil {
		defer pool.Close()
		if err := db.ApplySchemaAndSeed(ctx, pool, db.FindSQLDir()); err != nil {
			log.Printf("schema/seed: %v", err)
		}
		accounts = db.NewAccountRepo(pool)
		games = db.NewGameServerRepo(pool)
		log.Printf("login server using PostgreSQL")
	} else {
		log.Printf("PostgreSQL unavailable (%v), using in-memory stores", err)
	}

	srv, err := loginserver.NewServer(cfg, accounts, games)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting L2 Unity LoginServer (Go)")
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
