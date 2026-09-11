package main

import (
	"errors"
	"log"
	"math/rand/v2"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrNoRound      = errors.New("no round is active")
	ErrIsHolder     = errors.New("caller already holds the goat")
	ErrAlreadyEntry = errors.New("caller already entered this round")
	ErrRunning      = errors.New("a game is already running")
	ErrNoChannel    = errors.New("no announcement channel configured")
)

type Game struct {
	cfg   Config
	dg    Discord
	store Store
	rng   *rand.Rand
	s     *State

	reqs chan func()
}

func NewGame(cfg Config, dg Discord, store Store, s *State, rng *rand.Rand) *Game {
	return &Game{
		cfg:   cfg,
		dg:    dg,
		store: store,
		rng:   rng,
		s:     s,
		reqs:  make(chan func()),
	}
}

func (g *Game) Run(stop <-chan struct{}) {
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()

	g.maybeResolve()

	for {
		select {
		case fn := <-g.reqs:
			fn()
		case <-tick.C:
			g.maybeResolve()
		case <-stop:
			return
		}
	}
}

func (g *Game) do(fn func()) {
	done := make(chan struct{})
	g.reqs <- func() {
		defer close(done)
		fn()
	}
	<-done
}

func (g *Game) Steal(userID string) (data MsgData, err error) {
	g.do(func() { data, err = g.steal(userID) })
	return data, err
}

func (g *Game) Setup(targetID, fallbackChannel string) (data MsgData, err error) {
	g.do(func() { data, err = g.setup(targetID, fallbackChannel) })
	return data, err
}

func (g *Game) Reset(targetID, actorID string) (data MsgData, err error) {
	g.do(func() { data, err = g.reset(targetID, actorID) })
	return data, err
}

func (g *Game) ResolveNow() (err error) {
	g.do(func() { err = g.resolveNow() })
	return err
}

func (g *Game) Config(channelID string, minutes int) (data MsgData, err error) {
	g.do(func() { data, err = g.config(channelID, minutes) })
	return data, err
}

func (g *Game) StatusData() (started, unheld bool, data MsgData) {
	g.do(func() { started, unheld, data = g.statusData() })
	return started, unheld, data
}

func (g *Game) HistoryData() (reigns []Reign, total int) {
	g.do(func() { reigns, total = g.historyData() })
	return reigns, total
}

func (g *Game) EnsureRoleAtStartup() {
	g.do(g.ensureRoleAtStartup)
}

func (g *Game) save() {
	if err := g.store.Save(g.s); err != nil {
		log.Printf("state: save failed: %v", err)
	}
}

func (g *Game) startRound(now time.Time) {
	g.s.Challengers = nil
	g.s.RoundEndsAt = now.Add(g.s.roundDuration())
}

func (g *Game) memberPresent(userID string) bool {
	if userID == "" {
		return false
	}
	if _, err := g.dg.GuildMember(g.cfg.GuildID, userID); err != nil {
		if isNotFound(err) {
			return false
		}
		log.Printf("discord: member lookup for %s failed, assuming present: %v", userID, err)
	}
	return true
}

func (g *Game) maybeResolve() {
	if !g.s.started() || time.Now().Before(g.s.RoundEndsAt) {
		return
	}
	g.resolve()
}

func (g *Game) resolve() {
	now := time.Now()

	prevHolder := g.s.HolderID
	escaped := prevHolder != "" && !g.memberPresent(prevHolder)
	if escaped {
		g.s.recordReign(now)
		g.s.TotalTransfers++
	}

	var live []string
	for _, id := range g.s.Challengers {
		if g.memberPresent(id) {
			live = append(live, id)
		}
	}

	if len(live) == 0 {
		g.resolveUncontested(now, escaped, prevHolder)
		return
	}

	winner := live[g.rng.IntN(len(live))]

	if err := g.dg.GuildMemberRoleAdd(g.cfg.GuildID, winner, g.s.GoatRoleID); err != nil {
		log.Printf("discord: could not give the goat role to %s: %v", winner, err)
		g.startRound(now)
		g.save()
		g.announce(RoleFailedMsg, MsgData{
			Winner: mention(winner),
			Holder: mention(g.s.HolderID),
			Reason: err.Error(),
		})
		return
	}

	loser := g.s.HolderID
	if loser != "" {
		if err := g.dg.GuildMemberRoleRemove(g.cfg.GuildID, loser, g.s.GoatRoleID); err != nil {
			log.Printf("discord: could not take the goat role from %s, continuing anyway: %v", loser, err)
		}
		g.s.recordReign(now)
		g.s.TotalTransfers++
	}

	g.s.HolderID = winner
	g.s.HolderSince = now
	g.s.HolderStreak = 0

	challengers := live
	g.startRound(now)
	g.save()

	data := MsgData{
		Winner:      mention(winner),
		Challengers: mentionList(challengers),
		Count:       len(challengers),
		Deadline:    relTime(g.s.RoundEndsAt),
		DeadlineAt:  clockTime(g.s.RoundEndsAt),
		Total:       g.s.TotalTransfers,
	}
	if escaped {
		data.Loser = mention(prevHolder)
		g.announce(EscapedRecoveredMsg, data)
		return
	}
	data.Loser = mention(loser)
	g.announce(StolenMsg, data)
}

func (g *Game) resolveUncontested(now time.Time, escaped bool, prevHolder string) {
	if escaped {
		g.startRound(now)
		g.save()
		g.announce(EscapedUnheldMsg, MsgData{
			Loser:      mention(prevHolder),
			Deadline:   relTime(g.s.RoundEndsAt),
			DeadlineAt: clockTime(g.s.RoundEndsAt),
		})
		return
	}

	if g.s.HolderID == "" {
		g.startRound(now)
		g.save()
		return
	}

	g.s.HolderStreak++
	g.startRound(now)
	g.save()
	g.announce(SurvivedMsg, MsgData{
		Holder:     mention(g.s.HolderID),
		Streak:     g.s.HolderStreak,
		HeldSince:  relTime(g.s.HolderSince),
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
	})
}

func (g *Game) steal(userID string) (MsgData, error) {
	if !g.s.started() {
		return MsgData{}, ErrNoRound
	}
	if userID != "" && userID == g.s.HolderID {
		return MsgData{
			Streak:     g.s.HolderStreak,
			Deadline:   relTime(g.s.RoundEndsAt),
			DeadlineAt: clockTime(g.s.RoundEndsAt),
		}, ErrIsHolder
	}

	already := g.s.entered(userID)
	if !already {
		g.s.Challengers = append(g.s.Challengers, userID)
		g.save()
	}

	data := MsgData{
		Count:      len(g.s.Challengers),
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
	}
	if already {
		return data, ErrAlreadyEntry
	}
	return data, nil
}

func (g *Game) setup(targetID, fallbackChannel string) (MsgData, error) {
	if g.s.HolderID != "" {
		return MsgData{Holder: mention(g.s.HolderID)}, ErrRunning
	}
	if g.s.AnnounceChannel == "" {
		g.s.AnnounceChannel = fallbackChannel
	}
	if err := g.ensureRole(); err != nil {
		return MsgData{Reason: err.Error()}, err
	}
	if err := g.dg.GuildMemberRoleAdd(g.cfg.GuildID, targetID, g.s.GoatRoleID); err != nil {
		return MsgData{Reason: err.Error()}, err
	}

	now := time.Now()
	g.s.HolderID = targetID
	g.s.HolderSince = now
	g.s.HolderStreak = 0
	g.startRound(now)
	g.save()

	data := MsgData{
		Holder:     mention(targetID),
		Minutes:    g.s.RoundMinutes,
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
	}
	g.announce(ClaimedMsg, data)
	return data, nil
}

func (g *Game) reset(targetID, actorID string) (MsgData, error) {
	if err := g.ensureRole(); err != nil {
		return MsgData{Reason: err.Error()}, err
	}
	if err := g.dg.GuildMemberRoleAdd(g.cfg.GuildID, targetID, g.s.GoatRoleID); err != nil {
		return MsgData{Reason: err.Error()}, err
	}

	now := time.Now()
	if g.s.HolderID != "" && g.s.HolderID != targetID {
		if err := g.dg.GuildMemberRoleRemove(g.cfg.GuildID, g.s.HolderID, g.s.GoatRoleID); err != nil {
			log.Printf("discord: could not take the goat role from %s during reset: %v", g.s.HolderID, err)
		}
	}
	g.s.recordReign(now)
	g.s.HolderID = targetID
	g.s.HolderSince = now
	g.s.HolderStreak = 0
	g.startRound(now)
	g.save()

	data := MsgData{
		Holder:     mention(targetID),
		Actor:      mention(actorID),
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
	}
	g.announce(ResetMsg, data)
	return data, nil
}

func (g *Game) resolveNow() error {
	if !g.s.started() {
		return ErrNoRound
	}
	g.resolve()
	return nil
}

func (g *Game) config(channelID string, minutes int) (MsgData, error) {
	if channelID != "" {
		g.s.AnnounceChannel = channelID
	}
	if minutes > 0 {
		g.s.RoundMinutes = minutes
	}
	g.save()

	data := MsgData{Channel: channelRef(g.s.AnnounceChannel), Minutes: g.s.RoundMinutes}
	if g.s.AnnounceChannel == "" {
		return data, ErrNoChannel
	}
	return data, nil
}

func (g *Game) statusData() (started, unheld bool, data MsgData) {
	return g.s.started(), g.s.HolderID == "", MsgData{
		Holder:     mention(g.s.HolderID),
		Streak:     g.s.HolderStreak,
		Count:      len(g.s.Challengers),
		HeldSince:  relTime(g.s.HolderSince),
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
		Total:      g.s.TotalTransfers,
	}
}

func (g *Game) historyData() ([]Reign, int) {
	n := len(g.s.History)
	from := 0
	if n > historyShown {
		from = n - historyShown
	}
	return append([]Reign(nil), g.s.History[from:]...), g.s.TotalTransfers
}

func (g *Game) ensureRole() error {
	if g.s.GoatRoleID != "" {
		_, err := g.dg.GuildRole(g.cfg.GuildID, g.s.GoatRoleID)
		if err == nil {
			return nil
		}
		if !isNotFound(err) {
			return err
		}
		log.Printf("discord: goat role %s is gone, creating a new one", g.s.GoatRoleID)
	}

	name := goatRoleName
	color := goatRoleColor
	hoist := true
	mentionable := false
	role, err := g.dg.GuildRoleCreate(g.cfg.GuildID, &discordgo.RoleParams{
		Name:        &name,
		Color:       &color,
		Hoist:       &hoist,
		Mentionable: &mentionable,
	})
	if err != nil {
		return err
	}
	g.s.GoatRoleID = role.ID
	g.save()
	return nil
}

func (g *Game) ensureRoleAtStartup() {
	if g.s.GoatRoleID == "" {
		return
	}
	if err := g.ensureRole(); err != nil {
		log.Printf("discord: goat role unavailable: %v", err)
		return
	}
	if g.s.HolderID == "" || !g.memberPresent(g.s.HolderID) {
		return
	}
	if err := g.dg.GuildMemberRoleAdd(g.cfg.GuildID, g.s.HolderID, g.s.GoatRoleID); err != nil {
		log.Printf("discord: could not restore the goat role on %s: %v", g.s.HolderID, err)
	}
}
