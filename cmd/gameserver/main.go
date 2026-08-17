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
	var npcs []gameserver.NPC
	if pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 45*time.Second); err == nil {
		defer pool.Close()
		if err := db.ApplySchemaAndSeed(ctx, pool, db.FindSQLDir()); err != nil {
			log.Printf("schema/seed: %v", err)
		}
		store = db.NewCharacterRepo(pool)
		if loaded, err := db.NewNpcRepo(pool).ListSpawns(ctx); err == nil {
			npcs = loaded
		}
		log.Printf("game server using PostgreSQL")
		// Load XML first so PersistDatapack has rows.
		if err := gameserver.LoadDatapack(gameserver.FindDataDir()); err != nil {
			log.Printf("datapack: %v", err)
		} else if err := pool.PersistDatapack(ctx); err != nil {
			log.Printf("persist datapack: %v", err)
		} else {
			log.Printf("persisted %d skill levels and %d class trees", gameserver.SkillCount(), gameserver.ClassCount())
		}
	} else {
		log.Printf("PostgreSQL unavailable (%v), using in-memory store", err)
		store = gameserver.NewMemoryCharacterStore()
	}

	srv := gameserver.NewServer(cfg, store)
	if len(npcs) > 0 {
		srv.SeedWorld(npcs)
		log.Printf("loaded %d NPC spawns from database", len(npcs))
	}
	log.Printf("Starting L2 Unity GameServer (Go)")
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
