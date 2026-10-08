package bot

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
	cfg        Config
	dg         Discord
	store      Store
	rng        *rand.Rand
	retryDelay time.Duration
	pick       func(int) int
	s          *State

	reqs chan func()
}

func NewGame(cfg Config, dg Discord, store Store, s *State, rng *rand.Rand) *Game {
	g := &Game{
		cfg:        cfg,
		dg:         dg,
		store:      store,
		rng:        rng,
		retryDelay: 2 * time.Second,
		s:          s,
		reqs:       make(chan func()),
	}
	g.pick = g.rng.IntN
	return g
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

func (g *Game) Setup(fallbackChannel string) (data MsgData, err error) {
	g.do(func() { data, err = g.setup(fallbackChannel) })
	return data, err
}

func (g *Game) Reset(actorID string) (data MsgData, err error) {
	g.do(func() { data, err = g.reset(actorID) })
	return data, err
}

func (g *Game) ResolveNow() (err error) {
	g.do(func() { err = g.resolveNow() })
	return err
}

func (g *Game) Config(channelID, homeID string, minutes int) (data MsgData, err error) {
	g.do(func() { data, err = g.config(channelID, homeID, minutes) })
	return data, err
}

func (g *Game) ClearHome() (data MsgData) {
	g.do(func() { data = g.clearHome() })
	return data
}

func (g *Game) IsHolder(userID string) bool {
	var ok bool
	g.do(func() { ok = g.s.started() && g.s.HolderID == userID })
	return ok
}

func (g *Game) StatusData() (started, unheld bool, data MsgData) {
	g.do(func() { started, unheld, data = g.statusData() })
	return started, unheld, data
}

func (g *Game) HistoryData() (reigns []Reign, total, escapes, journeys int) {
	g.do(func() { reigns, total, escapes, journeys = g.historyData() })
	return reigns, total, escapes, journeys
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

func transient(err error) bool {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) {
		if rest.Response == nil {
			return true
		}
		return rest.Response.StatusCode >= 500
	}
	return true
}

func (g *Game) giveGoatRole(userID string) error {
	var err error
	for attempt := 1; attempt <= 4; attempt++ {
		err = g.dg.GuildMemberRoleAdd(g.cfg.GuildID, userID, g.s.GoatRoleID)
		if err == nil {
			return nil
		}
		if !transient(err) {
			return err
		}
		log.Printf("discord: giving the goat role to %s failed (attempt %d of 4), retrying: %v", userID, attempt, err)
		time.Sleep(time.Duration(attempt) * g.retryDelay)
	}
	return err
}

func (g *Game) takeGoatRole(userID string) error {
	var err error
	for attempt := 1; attempt <= 4; attempt++ {
		err = g.dg.GuildMemberRoleRemove(g.cfg.GuildID, userID, g.s.GoatRoleID)
		if err == nil {
			return nil
		}
		if !transient(err) {
			return err
		}
		log.Printf("discord: taking the goat role from %s failed (attempt %d of 4), retrying: %v", userID, attempt, err)
		time.Sleep(time.Duration(attempt) * g.retryDelay)
	}
	return err
}

func (g *Game) memberPresent(userID string) bool {
	if userID == "" {
		return false
	}
	if _, err := g.dg.GuildMember(g.cfg.GuildID, userID); err != nil {
		if isNotFound(err) {
			return false
		}
		log.Printf("discord: member lookup failed, assuming present: %v", err)
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

	draw := g.pick(len(live) + 1)
	if draw == len(live) {
		g.slipAway(now, live, escaped, prevHolder)
		return
	}
	winner := live[draw]

	if err := g.giveGoatRole(winner); err != nil {
		log.Printf("discord: could not give the goat role: %v", err)
		g.startRound(now)
		g.save()
		g.announce(RoleFailedMsg, MsgData{
			Winner:    mention(winner),
			Holder:    mention(g.s.HolderID),
			Count:     len(live),
			Transient: transient(err),
			Reason:    err.Error(),
		})
		return
	}

	loser := g.s.HolderID
	if loser != "" {
		if err := g.takeGoatRole(loser); err != nil {
			log.Printf("discord: could not take the goat role, continuing anyway: %v", err)
		}
		g.s.recordReign(now)
		g.s.TotalTransfers++
	}

	g.s.HolderID = winner
	g.s.HolderSince = now
	g.s.HolderStreak = 0
	g.s.UnheldSince = time.Time{}

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

func (g *Game) slipAway(now time.Time, live []string, escaped bool, prevHolder string) {
	loser := g.s.HolderID
	if loser != "" {
		if err := g.takeGoatRole(loser); err != nil {
			log.Printf("discord: could not take the goat role as it slipped away: %v", err)
		}
		g.s.recordReign(now)
	}
	g.s.TotalEscapes++
	if g.s.UnheldSince.IsZero() {
		g.s.UnheldSince = now
	}
	g.startRound(now)
	g.save()

	data := MsgData{
		Challengers: mentionList(live),
		Count:       len(live),
		Free:        heldFor(g.s.UnheldSince),
		Deadline:    relTime(g.s.RoundEndsAt),
		DeadlineAt:  clockTime(g.s.RoundEndsAt),
		Total:       g.s.TotalTransfers,
	}
	if escaped {
		data.Loser = mention(prevHolder)
	} else {
		data.Loser = mention(loser)
	}
	g.announce(SlippedAwayMsg, data)
}

func (g *Game) goHome(now time.Time) bool {
	if g.s.HomeID == "" || !g.memberPresent(g.s.HomeID) {
		return false
	}
	if g.s.UnheldSince.IsZero() {
		g.s.UnheldSince = now
	}
	free := heldFor(g.s.UnheldSince)

	if err := g.giveGoatRole(g.s.HomeID); err != nil {
		log.Printf("discord: could not send the goat home: %v", err)
		return false
	}

	g.s.HolderID = g.s.HomeID
	g.s.HolderSince = now
	g.s.HolderStreak = 0
	g.s.UnheldSince = time.Time{}
	g.s.TotalJourneysHome++
	g.startRound(now)
	g.save()

	g.announce(HomeMsg, MsgData{
		Holder:     mention(g.s.HolderID),
		Free:       free,
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
	})
	return true
}

func (g *Game) resolveUncontested(now time.Time, escaped bool, prevHolder string) {
	if escaped {
		g.s.TotalEscapes++
		if g.s.UnheldSince.IsZero() {
			g.s.UnheldSince = now
		}
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
		if g.goHome(now) {
			return
		}
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
		Held:       heldFor(g.s.HolderSince),
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

func (g *Game) setup(fallbackChannel string) (MsgData, error) {
	if g.s.started() {
		return MsgData{
			Holder:     mention(g.s.HolderID),
			Minutes:    g.s.RoundMinutes,
			Deadline:   relTime(g.s.RoundEndsAt),
			DeadlineAt: clockTime(g.s.RoundEndsAt),
		}, ErrRunning
	}
	if g.s.AnnounceChannel == "" {
		g.s.AnnounceChannel = fallbackChannel
	}
	if err := g.ensureRole(); err != nil {
		return MsgData{Reason: err.Error()}, err
	}

	now := time.Now()
	g.s.UnheldSince = now
	g.startRound(now)
	g.save()

	data := MsgData{
		Minutes:    g.s.RoundMinutes,
		Free:       heldFor(g.s.UnheldSince),
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
	}
	g.announce(ClaimedMsg, data)
	return data, nil
}

func (g *Game) reset(actorID string) (MsgData, error) {
	if err := g.ensureRole(); err != nil {
		return MsgData{Reason: err.Error()}, err
	}

	now := time.Now()
	if g.s.HolderID != "" {
		if err := g.takeGoatRole(g.s.HolderID); err != nil {
			log.Printf("discord: could not take the goat role during reset: %v", err)
		}
	}
	g.s.recordReign(now)
	g.s.UnheldSince = now
	g.startRound(now)
	g.save()

	data := MsgData{
		Actor:      mention(actorID),
		Minutes:    g.s.RoundMinutes,
		Free:       heldFor(g.s.UnheldSince),
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

func (g *Game) config(channelID, homeID string, minutes int) (MsgData, error) {
	if channelID != "" {
		g.s.AnnounceChannel = channelID
	}
	if homeID != "" {
		g.s.HomeID = homeID
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

func (g *Game) clearHome() MsgData {
	g.s.HomeID = ""
	g.save()
	return MsgData{Channel: channelRef(g.s.AnnounceChannel), Minutes: g.s.RoundMinutes}
}

func (g *Game) statusData() (started, unheld bool, data MsgData) {
	return g.s.started(), g.s.HolderID == "", MsgData{
		Holder:     mention(g.s.HolderID),
		Streak:     g.s.HolderStreak,
		Count:      len(g.s.Challengers),
		HeldSince:  relTime(g.s.HolderSince),
		Held:       heldFor(g.s.HolderSince),
		Free:       heldFor(g.s.UnheldSince),
		Deadline:   relTime(g.s.RoundEndsAt),
		DeadlineAt: clockTime(g.s.RoundEndsAt),
		Total:      g.s.TotalTransfers,
	}
}

func (g *Game) historyData() ([]Reign, int, int, int) {
	n := len(g.s.History)
	from := 0
	if n > historyShown {
		from = n - historyShown
	}
	return append([]Reign(nil), g.s.History[from:]...), g.s.TotalTransfers, g.s.TotalEscapes, g.s.TotalJourneysHome
}

func (g *Game) ensureRole() error {
	roles, err := g.dg.GuildRoles(g.cfg.GuildID)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r.ID == g.s.GoatRoleID && g.s.GoatRoleID != "" {
			return nil
		}
	}
	for _, r := range roles {
		if r.Name == goatRoleName {
			g.s.GoatRoleID = r.ID
			g.save()
			return nil
		}
	}
	if g.s.GoatRoleID != "" {
		log.Printf("discord: goat role %s is gone, creating a new one", g.s.GoatRoleID)
	}

	color := goatRoleColor
	hoist := true
	mentionable := false
	role, err := g.dg.GuildRoleCreate(g.cfg.GuildID, &discordgo.RoleParams{
		Name:        goatRoleName,
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
	if err := g.giveGoatRole(g.s.HolderID); err != nil {
		log.Printf("discord: could not restore the goat role: %v", err)
	}
}
