package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pblaravel/game-server-l2-go/internal/config"
	"github.com/pblaravel/game-server-l2-go/internal/db"
	"github.com/pblaravel/game-server-l2-go/internal/gameserver"
)

func main() {
	cfg, err := config.LoadGameConfig("conf/gameserver/server.properties", "conf/server.properties")
	if err != nil {
		log.Printf("using default game config: %v", err)
		cfg = config.DefaultGameConfig()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var store gameserver.CharacterStore
	if pool, err := db.Connect(ctx, cfg.DatabaseURL); err == nil {
		defer pool.Close()
		store = db.NewCharacterRepo(pool)
		log.Printf("game server using PostgreSQL")
	} else {
		log.Printf("PostgreSQL unavailable (%v), using in-memory store", err)
		store = gameserver.NewMemoryCharacterStore()
	}

	srv := gameserver.NewServer(cfg, store)
	log.Printf("Starting L2 Unity GameServer (Go)")
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
