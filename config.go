package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type Config struct {
	Token     string
	GuildID   string
	StatePath string
}

func loadConfig() (Config, error) {
	loadDotEnv(".env")

	c := Config{
		Token:     strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		GuildID:   strings.TrimSpace(os.Getenv("GUILD_ID")),
		StatePath: strings.TrimSpace(os.Getenv("STATE_PATH")),
	}
	if c.StatePath == "" {
		c.StatePath = "state.json"
	}
	if c.Token == "" {
		return c, errors.New("DISCORD_TOKEN is not set: put it in .env next to the binary, or in the environment")
	}
	if c.GuildID == "" {
		return c, errors.New("GUILD_ID is not set: enable Developer Mode in Discord, right-click the server, Copy Server ID")
	}
	return c, nil
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
