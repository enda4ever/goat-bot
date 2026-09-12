package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultRoundMinutes = 60
	historyCap          = 50
	historyShown        = 10
	goatRoleName        = "🐐 Goat Thief"
	goatRoleColor       = 0xC8A165
)

type Reign struct {
	UserID string    `json:"user_id"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Streak int       `json:"streak"`
}

type State struct {
	GoatRoleID      string    `json:"goat_role_id"`
	AnnounceChannel string    `json:"announce_channel"`
	RoundMinutes    int       `json:"round_minutes"`
	HolderID        string    `json:"holder_id"`
	HolderSince     time.Time `json:"holder_since"`
	HolderStreak    int       `json:"holder_streak"`
	RoundEndsAt     time.Time `json:"round_ends_at"`
	Challengers     []string  `json:"challengers"`
	History         []Reign   `json:"history"`
	TotalTransfers  int       `json:"total_transfers"`
}

func (s *State) started() bool {
	return !s.RoundEndsAt.IsZero()
}

func (s *State) entered(userID string) bool {
	for _, id := range s.Challengers {
		if id == userID {
			return true
		}
	}
	return false
}

func (s *State) roundDuration() time.Duration {
	if s.RoundMinutes <= 0 {
		return defaultRoundMinutes * time.Minute
	}
	return time.Duration(s.RoundMinutes) * time.Minute
}

func (s *State) recordReign(to time.Time) {
	if s.HolderID == "" {
		return
	}
	s.History = append(s.History, Reign{
		UserID: s.HolderID,
		From:   s.HolderSince,
		To:     to,
		Streak: s.HolderStreak,
	})
	if len(s.History) > historyCap {
		s.History = s.History[len(s.History)-historyCap:]
	}
	s.HolderID = ""
	s.HolderStreak = 0
	s.HolderSince = time.Time{}
}

type Store interface {
	Load() (*State, error)
	Save(*State) error
}

type FileStore struct {
	Path string
}

func (f FileStore) Load() (*State, error) {
	b, err := os.ReadFile(f.Path)
	if os.IsNotExist(err) {
		return &State{RoundMinutes: defaultRoundMinutes}, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.RoundMinutes <= 0 {
		s.RoundMinutes = defaultRoundMinutes
	}
	return &s, nil
}

func (f FileStore) Save(s *State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(f.Path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.Path)
}
