package main

import (
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	log.SetFlags(log.LstdFlags)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	store := FileStore{Path: cfg.StatePath}
	state, err := store.Load()
	if err != nil {
		log.Fatalf("state: could not read %s: %v", cfg.StatePath, err)
	}

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalf("discord: %v", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds

	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	game := NewGame(cfg, session, store, state, rng)

	session.AddHandler(handler{game: game}.route)
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("discord: connected as %s", r.User.String())
	})

	if err := session.Open(); err != nil {
		log.Fatalf("discord: could not connect: %v", err)
	}
	defer session.Close()

	stop := make(chan struct{})
	go game.Run(stop)

	game.EnsureRoleAtStartup()

	if _, err := session.ApplicationCommandBulkOverwrite(session.State.User.ID, cfg.GuildID, commandDefs()); err != nil {
		log.Printf("discord: could not register commands: %v", err)
	} else {
		log.Printf("discord: commands registered for guild %s", cfg.GuildID)
	}

	log.Printf("state: %s, rounds are %d minutes", cfg.StatePath, state.RoundMinutes)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	close(stop)
	log.Print("shutting down")
}
