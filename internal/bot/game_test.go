package bot

import (
	"github.com/bwmarrin/discordgo"
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
		&StolenMsg:           "STOLEN winner={{.Winner}} loser={{.Loser}} count={{.Count}} friend={{.Friend}}",
		&EscapedRecoveredMsg: "ESCAPED_RECOVERED winner={{.Winner}} loser={{.Loser}}",
		&EscapedUnheldMsg:    "ESCAPED_UNHELD loser={{.Loser}}",
		&ResetMsg:            "RESET {{.Holder}}",
		&RoleFailedMsg:       "ROLE_FAILED winner={{.Winner}} holder={{.Holder}}",
		&SlippedAwayMsg:      "SLIPPED_AWAY loser={{.Loser}} count={{.Count}}",
		&PetBrokeFreeMsg:     "PET_BROKE_FREE holder={{.Holder}} friend={{.Friend}}",
		&PetNoEffectMsg:      "PET_NO_EFFECT holder={{.Holder}} friend={{.Friend}}",
		&PetBonusMsg:         "PET_BONUS holder={{.Holder}} friend={{.Friend}}",
		&PetLoyalMsg:         "PET_LOYAL holder={{.Holder}}",
		&HomeMsg:             "HOME holder={{.Holder}} free={{.Free}}",
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
	game.retryDelay = 0
	return game, fake, store
}

func alwaysPicks(g *Game, i int) {
	g.pick = func(int) int { return i }
}

func baseState() *State {
	return &State{
		GoatRoleID:      "role-goat",
		FriendRoleID:    "role-friend",
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
	alwaysPicks(game, 0)

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

	slots := len(entrants) + 1
	want := rounds / slots
	tolerance := want / 5
	for _, e := range append(append([]string(nil), entrants...), "") {
		got := wins[e]
		if got == 0 {
			t.Errorf("%q never won in %d rounds", e, rounds)
			continue
		}
		if got < want-tolerance || got > want+tolerance {
			t.Errorf("%q won %d of %d, want roughly %d", e, got, rounds, want)
		}
	}
	if len(wins) != slots {
		t.Errorf("winners = %v, want the entrants plus nobody", wins)
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
	alwaysPicks(game, 0)
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
		if s.HolderID == "ghost" {
			t.Fatalf("round %d: a departed member won", i)
		}
	}
}

func TestDurationChangeAppliesToNextRound(t *testing.T) {
	s := baseState()
	deadline := time.Now().Add(45 * time.Minute)
	s.RoundEndsAt = deadline
	game, _, _ := newTestGame(t, s, "alice")

	if _, err := game.config("", "", 2); err != nil {
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

	if _, err := game.setup("chan"); err != ErrRunning {
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

func TestGoatSlipsAwayFromEverybody(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob", "carol"}
	game, fake, _ := newTestGame(t, s, "alice", "bob", "carol")
	alwaysPicks(game, 2)

	game.resolve()

	if s.HolderID != "" {
		t.Fatalf("holder = %q, want nobody", s.HolderID)
	}
	if s.UnheldSince.IsZero() {
		t.Error("UnheldSince was not stamped")
	}
	if s.TotalTransfers != 0 {
		t.Errorf("transfers = %d, want 0; nobody became a keeper", s.TotalTransfers)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 0 {
		t.Errorf("goat role held by %v, want nobody", got)
	}
	if len(s.History) != 1 || s.History[0].UserID != "alice" {
		t.Fatalf("history = %+v, want one reign for alice", s.History)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "SLIPPED_AWAY") {
		t.Errorf("announced %q, want the slipped-away message", got)
	}
	for _, m := range fake.sent {
		if strings.HasPrefix(m, "STOLEN") {
			t.Error("a theft was announced when nobody took the goat")
		}
	}
}

func TestUnheldSinceSurvivesFurtherRounds(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, _, _ := newTestGame(t, s, "alice", "bob")
	alwaysPicks(game, 1)

	game.resolve()
	first := s.UnheldSince
	if first.IsZero() {
		t.Fatal("UnheldSince was not stamped")
	}

	s.Challengers = []string{"bob"}
	s.RoundEndsAt = time.Now().Add(-time.Second)
	game.resolve()

	if !s.UnheldSince.Equal(first) {
		t.Errorf("UnheldSince = %v, want it unchanged at %v", s.UnheldSince, first)
	}
}

func TestSetupLeavesTheGoatUnheld(t *testing.T) {
	s := baseState()
	s.HolderID = ""
	s.HolderSince = time.Time{}
	s.RoundEndsAt = time.Time{}
	game, fake, _ := newTestGame(t, s, "alice")

	if _, err := game.setup("chan"); err != nil {
		t.Fatalf("setup err = %v", err)
	}
	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if s.UnheldSince.IsZero() {
		t.Error("UnheldSince was not stamped")
	}
	if !s.started() {
		t.Error("round did not start")
	}
	if fake.addCalls != 0 {
		t.Errorf("role add calls = %d, want none; setup assigns nobody", fake.addCalls)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "CLAIMED") {
		t.Errorf("announced %q, want the claimed message", got)
	}
}

func TestResetLeavesTheGoatUnheld(t *testing.T) {
	s := baseState()
	game, fake, _ := newTestGame(t, s, "alice", "bob")

	if _, err := game.reset("bob"); err != nil {
		t.Fatalf("reset err = %v", err)
	}
	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if s.UnheldSince.IsZero() {
		t.Error("UnheldSince was not stamped")
	}
	if len(s.History) != 1 || s.History[0].UserID != "alice" {
		t.Fatalf("history = %+v, want alice's reign recorded", s.History)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 0 {
		t.Errorf("goat role held by %v, want nobody", got)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "RESET") {
		t.Errorf("announced %q, want the reset message", got)
	}
}

func TestDepartedHolderIsNotCountedAsATheft(t *testing.T) {
	s := baseState()
	game, _, _ := newTestGame(t, s, "alice")
	fake := game.dg.(*fakeDiscord)
	fake.leave("alice")

	game.resolve()

	if s.TotalTransfers != 0 {
		t.Errorf("thefts = %d, want 0; nobody took the goat", s.TotalTransfers)
	}
	if s.TotalEscapes != 1 {
		t.Errorf("escapes = %d, want 1", s.TotalEscapes)
	}
}

func unheldState() *State {
	s := baseState()
	s.HolderID = ""
	s.HolderSince = time.Time{}
	s.UnheldSince = time.Now().Add(-2 * time.Hour)
	return s
}

func TestGoatGoesHomeAfterAFreeRound(t *testing.T) {
	s := unheldState()
	s.HomeID = "home"
	game, fake, _ := newTestGame(t, s, "home")

	game.resolve()

	if s.HolderID != "home" {
		t.Fatalf("holder = %q, want home", s.HolderID)
	}
	if s.TotalJourneysHome != 1 {
		t.Errorf("journeys home = %d, want 1", s.TotalJourneysHome)
	}
	if s.TotalTransfers != 0 || s.TotalEscapes != 0 {
		t.Errorf("transfers = %d, escapes = %d, want both 0", s.TotalTransfers, s.TotalEscapes)
	}
	if !s.UnheldSince.IsZero() {
		t.Error("UnheldSince should be cleared once the goat is held")
	}
	if got := fake.holdersOf("role-goat"); len(got) != 1 || got[0] != "home" {
		t.Errorf("goat role held by %v, want [home]", got)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "HOME") {
		t.Errorf("announced %q, want the home message", got)
	}
}

func TestGoatStaysFreeWithNoHomeConfigured(t *testing.T) {
	s := unheldState()
	game, fake, store := newTestGame(t, s, "alice")
	ends := s.RoundEndsAt

	game.resolve()

	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if len(fake.sent) != 0 {
		t.Errorf("announced %v, want silence", fake.sent)
	}
	if !s.RoundEndsAt.After(ends) {
		t.Error("round did not advance")
	}
	if store.saves == 0 {
		t.Error("state was not saved")
	}
	if s.TotalJourneysHome != 0 || s.TotalEscapes != 0 {
		t.Error("counters should be untouched")
	}
}

func TestGoatStaysFreeWhenHomeHasLeft(t *testing.T) {
	s := unheldState()
	s.HomeID = "ghost"
	game, fake, _ := newTestGame(t, s, "alice")

	game.resolve()

	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if s.TotalJourneysHome != 0 {
		t.Errorf("journeys home = %d, want 0", s.TotalJourneysHome)
	}
	if len(fake.sent) != 0 {
		t.Errorf("announced %v, want silence", fake.sent)
	}
}

func TestGoatStaysFreeWhenRoleAddFails(t *testing.T) {
	s := unheldState()
	s.HomeID = "home"
	game, fake, _ := newTestGame(t, s, "home")
	fake.failRoleAdd = errForced

	game.resolve()

	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if s.TotalJourneysHome != 0 {
		t.Errorf("journeys home = %d, want 0", s.TotalJourneysHome)
	}
	if fake.addCalls != 1 {
		t.Errorf("role add calls = %d, want 1", fake.addCalls)
	}
	if len(fake.sent) != 0 {
		t.Errorf("announced %v, want silence", fake.sent)
	}
}

func TestOneStealAttemptPreventsHomecoming(t *testing.T) {
	s := unheldState()
	s.HomeID = "home"
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "home", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob", s.HolderID)
	}
	if s.TotalJourneysHome != 0 {
		t.Errorf("journeys home = %d, want 0", s.TotalJourneysHome)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "STOLEN") {
		t.Errorf("announced %q, want the stolen message", got)
	}
}

func TestHomeKeeperCanBeStolenFrom(t *testing.T) {
	s := baseState()
	s.HolderID = "home"
	s.HomeID = "home"
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "home", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob", s.HolderID)
	}
	if len(s.History) != 1 || s.History[0].UserID != "home" {
		t.Fatalf("history = %+v, want the home's reign recorded", s.History)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "STOLEN") {
		t.Errorf("announced %q, want the stolen message", got)
	}
}

func TestClearHomeRemovesTheStoredID(t *testing.T) {
	s := unheldState()
	s.HomeID = "home"
	game, fake, _ := newTestGame(t, s, "home")

	game.clearHome()
	if s.HomeID != "" {
		t.Fatalf("HomeID = %q, want empty", s.HomeID)
	}

	game.resolve()

	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody; the home was cleared", s.HolderID)
	}
	if len(fake.sent) != 0 {
		t.Errorf("announced %v, want silence", fake.sent)
	}
}

func TestTransientRoleFailureIsRetried(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.flakyAddTimes = 2

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob; a 503 should be retried, not thrown away", s.HolderID)
	}
	if fake.addCalls != 3 {
		t.Errorf("role add attempts = %d, want 3", fake.addCalls)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "STOLEN") {
		t.Errorf("announced %q, want the stolen message", got)
	}
}

func TestPermissionFailureIsNotRetried(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.failRoleAdd = errForced

	game.resolve()

	if fake.addCalls != 1 {
		t.Errorf("role add attempts = %d, want 1; a 403 will never succeed on retry", fake.addCalls)
	}
	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice to keep it", s.HolderID)
	}
}

func friendlyState(t *testing.T) *State {
	t.Helper()
	s := baseState()
	s.History = []Reign{{
		UserID: "alice",
		From:   time.Now().Add(-5 * time.Hour),
		To:     time.Now().Add(-4 * time.Hour),
	}}
	return s
}

func TestPetRefusedForNonHolder(t *testing.T) {
	s := baseState()
	game, _, _ := newTestGame(t, s, "alice", "bob")

	if _, _, err := game.pet("bob"); err != ErrNotHolder {
		t.Fatalf("err = %v, want ErrNotHolder", err)
	}
	if s.PettedThisRound {
		t.Error("a refused pet should not consume the round's pet")
	}
}

func TestSecondPetRefusedForAFriend(t *testing.T) {
	s := friendlyState(t)
	game, _, _ := newTestGame(t, s, "alice")
	alwaysPicks(game, 0)

	if _, _, err := game.pet("alice"); err != nil {
		t.Fatalf("first pet: %v", err)
	}
	if _, _, err := game.pet("alice"); err != ErrAlreadyPetted {
		t.Fatalf("second pet err = %v, want ErrAlreadyPetted", err)
	}
	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice; a friend keeps the goat", s.HolderID)
	}
}

func TestSecondPetFreesTheGoatForNewcomers(t *testing.T) {
	s := baseState()
	game, fake, _ := newTestGame(t, s, "alice")
	fake.roles["alice"] = []string{"role-goat"}
	alwaysPicks(game, 1)

	if _, out, err := game.pet("alice"); err != nil || out != petNoEffect {
		t.Fatalf("first pet: out = %v, err = %v", out, err)
	}

	_, out, err := game.pet("alice")
	if err != nil {
		t.Fatalf("second pet: %v", err)
	}
	if out != petBrokeFree {
		t.Fatalf("outcome = %v, want petBrokeFree", out)
	}
	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 0 {
		t.Errorf("goat role held by %v, want nobody", got)
	}
}

func TestPetCanBreakTheGoatFree(t *testing.T) {
	s := baseState()
	game, fake, _ := newTestGame(t, s, "alice")
	fake.roles["alice"] = []string{"role-goat"}
	alwaysPicks(game, 0)
	ends := s.RoundEndsAt

	_, out, err := game.pet("alice")
	if err != nil {
		t.Fatalf("pet: %v", err)
	}
	if out != petBrokeFree {
		t.Fatalf("outcome = %v, want petBrokeFree", out)
	}
	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 0 {
		t.Errorf("goat role held by %v, want nobody", got)
	}
	if len(s.History) != 1 || s.History[0].UserID != "alice" {
		t.Fatalf("history = %+v, want alice's reign", s.History)
	}
	if s.TotalEscapes != 1 {
		t.Errorf("escapes = %d, want 1", s.TotalEscapes)
	}
	if s.UnheldSince.IsZero() {
		t.Error("UnheldSince was not stamped")
	}
	if !s.RoundEndsAt.Equal(ends) {
		t.Error("the round should continue, not restart")
	}
}

func TestPetNoEffectLeavesTheRoundAlone(t *testing.T) {
	s := baseState()
	game, _, _ := newTestGame(t, s, "alice")
	alwaysPicks(game, 1)
	ends := s.RoundEndsAt

	_, out, err := game.pet("alice")
	if err != nil {
		t.Fatalf("pet: %v", err)
	}
	if out != petNoEffect {
		t.Fatalf("outcome = %v, want petNoEffect", out)
	}
	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice", s.HolderID)
	}
	if s.BonusRounds != 0 {
		t.Errorf("bonus rounds = %d, want 0", s.BonusRounds)
	}
	if !s.RoundEndsAt.Equal(ends) {
		t.Error("deadline moved")
	}
}

func TestPetGrantsABonusRound(t *testing.T) {
	s := baseState()
	game, _, _ := newTestGame(t, s, "alice")
	alwaysPicks(game, 2)

	_, out, err := game.pet("alice")
	if err != nil {
		t.Fatalf("pet: %v", err)
	}
	if out != petBonusRound {
		t.Fatalf("outcome = %v, want petBonusRound", out)
	}
	if s.BonusRounds != 1 {
		t.Errorf("bonus rounds = %d, want 1", s.BonusRounds)
	}
	if s.HolderID != "alice" {
		t.Errorf("holder = %q, want alice", s.HolderID)
	}
}

func TestBonusRoundKeepsHolderAndDiscardsChallengers(t *testing.T) {
	s := baseState()
	s.BonusRounds = 1
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if s.HolderID != "alice" {
		t.Fatalf("holder = %q, want alice kept the goat", s.HolderID)
	}
	if s.BonusRounds != 0 {
		t.Errorf("bonus rounds = %d, want spent", s.BonusRounds)
	}
	if len(s.Challengers) != 0 {
		t.Errorf("challengers = %v, want discarded", s.Challengers)
	}
	if s.HolderStreak != 1 {
		t.Errorf("streak = %d, want 1; a protected round still counts", s.HolderStreak)
	}
	if s.TotalTransfers != 0 {
		t.Errorf("transfers = %d, want 0", s.TotalTransfers)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "PET_LOYAL") {
		t.Errorf("announced %q, want the loyal message", got)
	}
}

func TestBonusRoundDoesNotRescueADepartedHolder(t *testing.T) {
	s := baseState()
	s.BonusRounds = 1
	game, fake, _ := newTestGame(t, s, "alice")
	fake.leave("alice")

	game.resolve()

	if s.HolderID != "" {
		t.Errorf("holder = %q, want nobody", s.HolderID)
	}
	if s.BonusRounds != 1 {
		t.Errorf("bonus rounds = %d, want untouched", s.BonusRounds)
	}
}

func TestFriendNeverBreaksFree(t *testing.T) {
	for _, pick := range []int{0, 1} {
		s := friendlyState(t)
		game, _, _ := newTestGame(t, s, "alice")
		alwaysPicks(game, pick)

		_, out, err := game.pet("alice")
		if err != nil {
			t.Fatalf("pick %d: pet: %v", pick, err)
		}
		if out == petBrokeFree {
			t.Errorf("pick %d: a friend lost the goat to petting", pick)
		}
		if s.HolderID != "alice" {
			t.Errorf("pick %d: holder = %q, want alice", pick, s.HolderID)
		}
	}
}

func TestFriendRoleGivenOnReturn(t *testing.T) {
	s := baseState()
	s.History = []Reign{{
		UserID: "bob",
		From:   time.Now().Add(-5 * time.Hour),
		To:     time.Now().Add(-4 * time.Hour),
	}}
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob", s.HolderID)
	}
	if got := fake.holdersOf("role-friend"); len(got) != 1 || got[0] != "bob" {
		t.Errorf("friend role held by %v, want [bob]", got)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 1 || got[0] != "bob" {
		t.Errorf("goat role held by %v, want [bob]", got)
	}
}

func TestFriendRoleWithheldOutsideWindow(t *testing.T) {
	s := baseState()
	s.History = []Reign{{
		UserID: "bob",
		From:   time.Now().Add(-26 * time.Hour),
		To:     time.Now().Add(-25 * time.Hour),
	}}
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if got := fake.holdersOf("role-friend"); len(got) != 0 {
		t.Errorf("friend role held by %v, want nobody; the reign is too old", got)
	}
}

func TestFriendRoleRemovedWhenGoatLeaves(t *testing.T) {
	s := friendlyState(t)
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.roles["alice"] = []string{"role-goat", "role-friend"}
	alwaysPicks(game, 0)

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob", s.HolderID)
	}
	for _, role := range []string{"role-goat", "role-friend"} {
		for _, who := range fake.holdersOf(role) {
			if who == "alice" {
				t.Errorf("alice still holds %s after losing the goat", role)
			}
		}
	}
}

func TestFriendRoleFailureStillCommitsTheTransfer(t *testing.T) {
	s := baseState()
	s.History = []Reign{{
		UserID: "bob",
		From:   time.Now().Add(-5 * time.Hour),
		To:     time.Now().Add(-4 * time.Hour),
	}}
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	fake.failAddOfRole = "role-friend"
	alwaysPicks(game, 0)

	game.resolve()

	if s.HolderID != "bob" {
		t.Fatalf("holder = %q, want bob; a cosmetic badge must not abort a theft", s.HolderID)
	}
	if got := fake.holdersOf("role-goat"); len(got) != 1 || got[0] != "bob" {
		t.Errorf("goat role held by %v, want [bob]", got)
	}
	if got := lastSent(fake); !strings.HasPrefix(got, "STOLEN") {
		t.Errorf("announced %q, want the stolen message", got)
	}
}

func TestStolenMessageNamesAReturningFriend(t *testing.T) {
	s := baseState()
	s.History = []Reign{{
		UserID: "bob",
		From:   time.Now().Add(-5 * time.Hour),
		To:     time.Now().Add(-4 * time.Hour),
	}}
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if got := lastSent(fake); !strings.Contains(got, "friend=true") {
		t.Errorf("announced %q, want the friend flag set", got)
	}
}

func TestStolenMessageOmitsFriendForNewcomers(t *testing.T) {
	s := baseState()
	s.Challengers = []string{"bob"}
	game, fake, _ := newTestGame(t, s, "alice", "bob")
	alwaysPicks(game, 0)

	game.resolve()

	if got := lastSent(fake); !strings.Contains(got, "friend=false") {
		t.Errorf("announced %q, want the friend flag unset", got)
	}
}

func TestPronounsDefaultToThey(t *testing.T) {
	s := baseState()
	game, fake, _ := newTestGame(t, s, "alice")
	if got := game.pronounsFor("alice").They(); got != "they" {
		t.Errorf("no roles configured: got %q, want they", got)
	}

	s.PronounRoles = map[string]string{"she": "role-she"}
	fake.roles["alice"] = []string{"role-unrelated"}
	if got := game.pronounsFor("alice").They(); got != "they" {
		t.Errorf("no pronoun role held: got %q, want they", got)
	}
}

func TestPronounsFromAMappedRole(t *testing.T) {
	s := baseState()
	s.PronounRoles = map[string]string{"she": "role-she", "he": "role-he"}
	game, fake, _ := newTestGame(t, s, "alice")
	fake.roles["alice"] = []string{"role-she"}

	p := game.pronounsFor("alice")
	if p.They() != "she" || p.Them() != "her" || p.Their() != "her" || p.TheyHave() != "she's" {
		t.Errorf("got %q/%q/%q/%q, want she/her/her/she's", p.They(), p.Them(), p.Their(), p.TheyHave())
	}
}

func TestComboRolePicksFromBothPronouns(t *testing.T) {
	for pick, want := range []string{"they", "she"} {
		s := baseState()
		s.PronounRoles = map[string]string{"theyshe": "role-theyshe"}
		game, fake, _ := newTestGame(t, s, "alice")
		fake.roles["alice"] = []string{"role-theyshe"}
		alwaysPicks(game, pick)

		if got := game.pronounsFor("alice").They(); got != want {
			t.Errorf("pick %d: got %q, want %q", pick, got, want)
		}
	}
}

func TestSeveralPronounRolesArePooled(t *testing.T) {
	for pick, want := range []string{"she", "he"} {
		s := baseState()
		s.PronounRoles = map[string]string{"she": "role-she", "he": "role-he"}
		game, fake, _ := newTestGame(t, s, "alice")
		fake.roles["alice"] = []string{"role-she", "role-he"}
		alwaysPicks(game, pick)

		if got := game.pronounsFor("alice").They(); got != want {
			t.Errorf("pick %d: got %q, want %q", pick, got, want)
		}
	}
}

func TestPronounsForADepartedMemberFallBack(t *testing.T) {
	s := baseState()
	s.PronounRoles = map[string]string{"she": "role-she"}
	game, _, _ := newTestGame(t, s, "alice")

	if got := game.pronounsFor("ghost").They(); got != "they" {
		t.Errorf("got %q, want they", got)
	}
}

func TestConfigPronounsOnlySetsGivenRoles(t *testing.T) {
	s := baseState()
	s.PronounRoles = map[string]string{"she": "role-she"}
	game, _, _ := newTestGame(t, s, "alice")

	game.configPronouns(map[string]string{"he": "role-he"})

	if s.PronounRoles["she"] != "role-she" || s.PronounRoles["he"] != "role-he" {
		t.Errorf("roles = %v, want she kept and he added", s.PronounRoles)
	}
}

func TestClearPronouns(t *testing.T) {
	s := baseState()
	s.PronounRoles = map[string]string{"she": "role-she"}
	game, _, _ := newTestGame(t, s, "alice")

	game.clearPronouns()

	if len(s.PronounRoles) != 0 {
		t.Errorf("roles = %v, want none", s.PronounRoles)
	}
}

func TestShippedCopyRendersPronouns(t *testing.T) {
	status := render(StatusMsg, MsgData{Holder: "@alice", Held: "3 hours", HolderPronouns: Pronouns{set: "she"}})
	if !strings.Contains(status, "She's held the creature") || !strings.Contains(status, "But she must") {
		t.Errorf("status = %q", status)
	}

	stolen := render(StolenMsg, MsgData{Winner: "@bob", Friend: true, WinnerPronouns: Pronouns{set: "he"}})
	if !strings.Contains(stolen, "counts him a") {
		t.Errorf("stolen = %q", stolen)
	}

	fallback := render(PetNoEffectMsg, MsgData{Holder: "@alice"})
	if !strings.Contains(fallback, "extended their hand") {
		t.Errorf("an unset pronoun should read they/them: %q", fallback)
	}
}

func TestPronounsAreOnlyFetchedWhenTheCopyUsesThem(t *testing.T) {
	s := baseState()
	s.PronounRoles = map[string]string{"she": "role-she"}
	game, fake, _ := newTestGame(t, s, "alice")
	fake.roles["alice"] = []string{"role-she"}

	before := fake.memberCalls
	render(ShowGoatMsg, MsgData{Holder: "@alice", HolderPronouns: game.pronounsFor("alice")})
	if fake.memberCalls != before {
		t.Errorf("a message without pronouns fetched the member %d times", fake.memberCalls-before)
	}

	before = fake.memberCalls
	out := render(StatusMsg, MsgData{Holder: "@alice", HolderPronouns: game.pronounsFor("alice")})
	if fake.memberCalls-before != 1 {
		t.Errorf("fetched %d times, want exactly once for several pronouns in one message", fake.memberCalls-before)
	}
	if !strings.Contains(out, "She's held") || !strings.Contains(out, "But she must") {
		t.Errorf("status = %q", out)
	}
}

func testRoles() []*discordgo.Role {
	return []*discordgo.Role{
		{ID: "g", Name: "@everyone"},
		{ID: "900", Name: "Goat Bot Test", Managed: true},
		{ID: "109", Name: "𝔓𝔢𝔞𝔰𝔞𝔫𝔱", Position: 9},
		{ID: "103", Name: "𝔰𝔥𝔢/𝔥𝔢𝔯", Position: 3},
		{ID: "102", Name: "𝔥𝔢/𝔰𝔥𝔢", Position: 2},
		{ID: "105", Name: "𝔱𝔥𝔢𝔶/𝔱𝔥𝔢𝔪", Position: 5},
	}
}

func choiceIDs(cs []*discordgo.ApplicationCommandOptionChoice) []string {
	ids := make([]string, len(cs))
	for i, c := range cs {
		ids[i], _ = c.Value.(string)
	}
	return ids
}

func TestPlainTextFlattensFancyFonts(t *testing.T) {
	for in, want := range map[string]string{
		"𝔰𝔥𝔢/𝔥𝔢𝔯": "she/her",
		"𝔥𝔢/𝔰𝔥𝔢":  "he/she",
		"𝕋𝕙𝕖𝕪/ℍ𝕖": "they/he",
	} {
		if got := plainText(in); got != want {
			t.Errorf("plainText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRoleSuggestionsPutPronounRolesFirst(t *testing.T) {
	got := choiceIDs(roleSuggestions(testRoles(), "g", ""))
	want := []string{"105", "103", "102", "109"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v: slashed roles first by position, no @everyone, no bot roles", got, want)
	}
}

func TestRoleSuggestionsMatchThroughFonts(t *testing.T) {
	got := choiceIDs(roleSuggestions(testRoles(), "g", "she"))
	if strings.Join(got, ",") != "103,102" {
		t.Errorf("typing she: got %v, want [she heshe]", got)
	}
}

func TestResolveRoleValue(t *testing.T) {
	roles := testRoles()
	cases := []struct {
		value, want string
		ok          bool
	}{
		{"she", "", false},
		{"he/she", "102", true},
		{"she/her", "103", true},
		{"they", "105", true},
		{"103", "103", true},
		{"nonsense", "", false},
		{"Goat Bot Test", "", false},
	}
	for _, c := range cases {
		got, ok := resolveRoleValue(roles, "g", c.value)
		if got != c.want || ok != c.ok {
			t.Errorf("resolveRoleValue(%q) = %q, %v; want %q, %v", c.value, got, ok, c.want, c.ok)
		}
	}
}

func TestAdminCommandsAreHiddenFromEveryoneElse(t *testing.T) {
	subs := map[string]map[string]bool{}
	var adminPerms *int64
	for _, c := range CommandDefs() {
		subs[c.Name] = map[string]bool{}
		for _, o := range c.Options {
			subs[c.Name][o.Name] = true
		}
		if c.Name == "goat-admin" {
			adminPerms = c.DefaultMemberPermissions
		}
	}
	if adminPerms == nil || *adminPerms != discordgo.PermissionManageServer {
		t.Fatalf("goat-admin permissions = %v, want Manage Server", adminPerms)
	}
	for _, name := range []string{"setup", "reset", "resolve", "config", "clear-home", "pronouns", "clear-pronouns", "preview"} {
		if !subs["goat-admin"][name] {
			t.Errorf("%s is missing from goat-admin", name)
		}
		if subs["goat"][name] {
			t.Errorf("%s is still visible to everyone under goat", name)
		}
	}
	for _, name := range []string{"steal", "pet", "status", "history", "show"} {
		if !subs["goat"][name] {
			t.Errorf("%s is missing from goat", name)
		}
	}
}
