package main

import (
	"math/rand/v2"
	"strings"
	"testing"
	"text/template"
	"time"
)

// The templates in strings.go are empty until the copy is written, and an empty
// slot is never sent. So the tests install their own marker copy: every
// assertion below stays independent of whatever text ends up shipping, and
// editing strings.go can never break the suite.
func withTestCopy(t *testing.T) {
	t.Helper()

	slots := map[**template.Template]string{
		&ClaimedMsg:          "CLAIMED {{.Holder}}",
		&SurvivedMsg:         "SURVIVED {{.Holder}} streak={{.Streak}}",
		&StolenMsg:           "STOLEN winner={{.Winner}} loser={{.Loser}} count={{.Count}}",
		&EscapedRecoveredMsg: "ESCAPED_RECOVERED winner={{.Winner}} loser={{.Loser}}",
		&EscapedUnheldMsg:    "ESCAPED_UNHELD loser={{.Loser}}",
		&ResetMsg:            "RESET {{.Holder}}",
		&RoleFailedMsg:       "ROLE_FAILED winner={{.Winner}} holder={{.Holder}}",
	}

	saved := make(map[**template.Template]*template.Template, len(slots))
	for p, body := range slots {
		saved[p] = *p
		*p = msg("test", body)
	}
	t.Cleanup(func() {
		for p, v := range saved {
			*p = v
		}
	})
}

// Tests drive the synchronous inner methods rather than the channel-based
// exported ones, so there is no second goroutine touching State and no race.
func newTestGame(t *testing.T, s *State, members ...string) (*Game, *fakeDiscord, *memStore) {
	t.Helper()
	withTestCopy(t)

	fake := newFake(members...)
	store := &memStore{}
	game := NewGame(Config{GuildID: "g", StatePath: "mem"}, fake, store, s, rand.New(rand.NewPCG(1, 2)))
	return game, fake, store
}

func baseState() *State {
	return &State{
		GoatRoleID:      "role-goat",
		AnnounceChannel: "chan",
		RoundMinutes:    60,
		HolderID:        "alice",
		HolderSince:     time.Now().Add(-30 * time.Minute),
		RoundEndsAt:     time.Now().Add(-time.Second),
	}
}

func lastSent(f *fakeDiscord) string {
	if len(f.sent) == 0 {
		return ""
	}
	return f.sent[len(f.sent)-1]
}

func TestSurvivesWhenNobodyEnters(t *testing.T) {
	s := baseState()
	game, fake, store := newTestGame(t, s, "alice", "bob")

	game.resolve()

	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice", s.HolderID)
	}
	if s.HolderStreak != 1 {
		t.Errorf("streak = %d, want 1", s.HolderStreak)
	}
	if fake.addCalls != 0 || fake.removeCalls != 0 {
		t.Errorf("role calls add=%d remove=%d, want none", fake.addCalls, fake.removeCalls)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "SURVIVED") {
		t.Errorf("announced %q, want the survival message", got)
	}
	if s.TotalTransfers != 0 {
		t.Errorf("transfers = %d, want 0", s.TotalTransfers)
	}
	if store.saves == 0 {
		t.Error("state was never saved")
	}
	if !s.RoundEndsAt.After(time.Now()) {
		t.Error("a fresh round should be open")
	}
}

func TestSingleChallengerWins(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob", s.HolderID)
	}
	if s.TotalTransfers != 1 {
		t.Errorf("transfers = %d, want 1", s.TotalTransfers)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 1 || got[0] != "bob" {
		t.Errorf("goat role held by %v, want [bob]", got)
	}
	if len(s.History) != 1 || s.History[0].UserID != "alice" {
		t.Fatalf("history = %+v, want one reign for alice", s.History)
	}
	if len(s.Challengers) != 0 {
		t.Errorf("challengers = %v, want cleared", s.Challengers)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "STOLEN") {
		t.Errorf("announced %q, want the stolen message", got)
	}
}

func TestStreakCountsRoundsSurvived(t *testing.T) {
	s := baseState()
	s.HolderStreak = 0
	game, _, _ := newTestGame(t, s, "alice", "bob")

	for round := 1; round <= 2; round++ {
		game.resolve()
		if s.HolderStreak != round {
			t.Fatalf("after %d survived rounds streak = %d", round, s.HolderStreak)
		}
	}

	s.Challengers = []string{"bob"}
	game.resolve()

	if len(s.History) != 1 {
		t.Fatalf("history = %+v, want one entry", s.History)
	}
	if s.History[0].Streak != 2 {
		t.Errorf("recorded streak = %d, want 2", s.History[0].Streak)
	}
	if s.HolderStreak != 0 {
		t.Errorf("new holder streak = %d, want 0", s.HolderStreak)
	}
}

func TestRoleAddFailureCommitsNothing(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.failRoleAdd = errForced

	game.resolve()

	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice to keep it", s.HolderID)
	}
	if s.TotalTransfers != 0 {
		t.Errorf("transfers = %d, want 0", s.TotalTransfers)
	}
	if len(s.History) != 0 {
		t.Errorf("history = %+v, want empty", s.History)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "ROLE_FAILED") {
		t.Fatalf("announced %q, want the role-failure warning", got)
	}
	for _, m := range fake.sent {
		if strings.HasPrefix(m, "STOLEN") {
			t.Error("a theft was announced even though the role never moved")
		}
	}
}

func TestRoleRemoveFailureStillCommits(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.failRoleRemove = errForced

	game.resolve()

	if s.HolderID != "bob" {
		t.Errorf("holder = %q, want bob", s.HolderID)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "STOLEN") {
		t.Errorf("announced %q, want the stolen message", got)
	}
}

func TestLotteryIsFair(t *testing.T) {
	const rounds = 10000
	entrants := []string{"a", "b", "c", "d"}

	s := baseState()
	game, _, _ := newTestGame(t, s, "holder", "a", "b", "c", "d")

	wins := map[string]int{}
	for i := 0; i < rounds; i++ {
		s.HolderID = "holder"
		s.HolderSince = time.Now()
		s.HolderStreak = 0
		s.History = nil
		s.Challengers = append([]string(nil), entrants...)

		game.resolve()
		wins[s.HolderID]++
	}

	want := rounds / len(entrants)
	tolerance := want / 5
	for _, e := range entrants {
		got := wins[e]
		if got == 0 {
			t.Errorf("%s never won in %d rounds", e, rounds)
			continue
		}
		if got < want-tolerance || got > want+tolerance {
			t.Errorf("%s won %d of %d, want roughly %d", e, got, rounds, want)
		}
	}
	if len(wins) != len(entrants) {
		t.Errorf("winners = %v, want exactly the entrants", wins)
	}
}

func TestStealRules(t *testing.T) {
	s := baseState()
	s.RoundEndsAt = time.Now().Add(time.Hour)
	game, _, _ := newTestGame(t, s, "alice", "bob")

	if _, err := game.steal("bob"); err != nil {
		t.Fatalf("first steal: %v", err)
	}
	if len(s.Challengers) != 1 {
		t.Fatalf("challengers = %v, want one entry", s.Challengers)
	}

	if _, err := game.steal("bob"); err != ErrAlreadyEntry {
		t.Errorf("second steal err = %v, want ErrAlreadyEntry", err)
	}
	if len(s.Challengers) != 1 {
		t.Errorf("challengers = %v, want still one entry", s.Challengers)
	}

	if _, err := game.steal("alice"); err != ErrIsHolder {
		t.Errorf("holder steal err = %v, want ErrIsHolder", err)
	}

	s.RoundEndsAt = time.Time{}
	if _, err := game.steal("bob"); err != ErrNoRound {
		t.Errorf("steal with no round err = %v, want ErrNoRound", err)
	}
}

func TestHolderLeftAndRecovered(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.leave("alice")

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob", s.HolderID)
	}
	if s.HolderStreak != 0 {
		t.Errorf("streak = %d, want 0", s.HolderStreak)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "ESCAPED_RECOVERED") {
		t.Errorf("announced %q, want the recovery message", got)
	}
	for _, m := range fake.sent {
		if strings.HasPrefix(m, "STOLEN") {
			t.Error("bob was credited with a theft against someone who left")
		}
	}
}

func TestHolderLeftWithNoChallengers(t *testing.T) {
	s := baseState()
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.leave("alice")

	game.resolve()

	if s.HolderID != "" {
		t.Errorf("holder = %q, want the goat unheld", s.HolderID)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "ESCAPED_UNHELD") {
		t.Errorf("announced %q, want the escaped message", got)
	}

	s.Challengers = []string{"bob"}
	game.resolve()
	if s.HolderID != "bob" {
		t.Errorf("holder = %q, want bob to pick up the loose goat", s.HolderID)
	}
}

func TestDepartedChallengerIsPruned(t *testing.T) {
	s := baseState()
	game, _, _ := newTestGame(t, s, "alice", "bob")

	for i := 0; i < 50; i++ {
		s.HolderID = "alice"
		s.Challengers = []string{"ghost", "bob"}

		game.resolve()
		if s.HolderID != "bob" {
			t.Fatalf("round %d: holder = %q, want bob; a departed member won", i, s.HolderID)
		}
	}
}

func TestDurationChangeAppliesToNextRound(t *testing.T) {
	s := baseState()
	deadline := time.Now().Add(45 * time.Minute)
	s.RoundEndsAt = deadline
	game, _, _ := newTestGame(t, s, "alice")

	if _, err := game.config("", 2); err != nil {
		t.Fatalf("config: %v", err)
	}
	if !s.RoundEndsAt.Equal(deadline) {
		t.Errorf("deadline moved to %v, want it left at %v", s.RoundEndsAt, deadline)
	}

	game.resolve()
	if got := time.Until(s.RoundEndsAt); got > 3*time.Minute {
		t.Errorf("next round runs for %v, want about 2 minutes", got)
	}
}

func TestSetupRefusesWhenRunning(t *testing.T) {
	s := baseState()
	game, fake, _ := newTestGame(t, s, "alice", "bob")

	if _, err := game.setup("bob", "chan"); err != ErrRunning {
		t.Fatalf("setup err = %v, want ErrRunning", err)
	}
	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice untouched", s.HolderID)
	}
	if fake.addCalls != 0 {
		t.Errorf("role add calls = %d, want none", fake.addCalls)
	}
}

func TestMissedRoundResolvesOnceOnRestart(t *testing.T) {
	s := baseState()
	s.RoundEndsAt = time.Now().Add(-3 * time.Hour)
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")

	game.maybeResolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want the missed round resolved to bob", s.HolderID)
	}
	if s.TotalTransfers != 1 {
		t.Errorf("transfers = %d, want exactly one, not one per missed hour", s.TotalTransfers)
	}
	if len(s.History) != 1 {
		t.Errorf("history has %d entries, want 1", len(s.History))
	}
	if got := time.Until(s.RoundEndsAt); got < 55*time.Minute {
		t.Errorf("next round ends in %v, want a full fresh round from now", got)
	}
	if n := len(fake.sent); n != 1 {
		t.Errorf("sent %d announcements, want 1", n)
	}

	game.maybeResolve()
	if s.TotalTransfers != 1 {
		t.Errorf("a second tick inside the round changed transfers to %d", s.TotalTransfers)
	}
}
