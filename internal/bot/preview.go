package bot

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

const discordLimit = 2000

const (
	DescPreview    = "Post every banner message here to check how they look (admin)"
	DescPreviewAll = "Also include the messages that have no banner"
)

type Message struct {
	Name string
	Tmpl *template.Template
}

func AllMessages() []Message {
	return []Message{
		{"ClaimedMsg", ClaimedMsg},
		{"SurvivedMsg", SurvivedMsg},
		{"StolenMsg", StolenMsg},
		{"EscapedRecoveredMsg", EscapedRecoveredMsg},
		{"SlippedAwayMsg", SlippedAwayMsg},
		{"EscapedUnheldMsg", EscapedUnheldMsg},
		{"HomeMsg", HomeMsg},
		{"ResetMsg", ResetMsg},
		{"PetBrokeFreeMsg", PetBrokeFreeMsg},
		{"PetNoEffectMsg", PetNoEffectMsg},
		{"PetBonusMsg", PetBonusMsg},
		{"PetLoyalMsg", PetLoyalMsg},
		{"EnteredMsg", EnteredMsg},
		{"AlreadyEnteredMsg", AlreadyEnteredMsg},
		{"YouHoldItMsg", YouHoldItMsg},
		{"AlreadyPettedMsg", AlreadyPettedMsg},
		{"NoRoundMsg", NoRoundMsg},
		{"NotAdminMsg", NotAdminMsg},
		{"AlreadyRunningMsg", AlreadyRunningMsg},
		{"ConfigDoneMsg", ConfigDoneMsg},
		{"PronounsSetMsg", PronounsSetMsg},
		{"PronounsUnknownMsg", PronounsUnknownMsg},
		{"StatusMsg", StatusMsg},
		{"StatusUnheldMsg", StatusUnheldMsg},
		{"HistoryHeaderMsg", HistoryHeaderMsg},
		{"HistoryDayMsg", HistoryDayMsg},
		{"HistoryLineMsg", HistoryLineMsg},
		{"HistoryEmptyMsg", HistoryEmptyMsg},
		{"ShowGoatMsg", ShowGoatMsg},
		{"ShowNotHolderMsg", ShowNotHolderMsg},
		{"RoleFailedMsg", RoleFailedMsg},
		{"ChannelMissingMsg", ChannelMissingMsg},
		{"RoleMissingMsg", RoleMissingMsg},
	}
}

func (g *Game) Preview(channelID string, all bool) string {
	odd := map[rune]bool{}
	var posted, skipped int
	var problems []string

	for _, m := range AllMessages() {
		data := previewData()
		body := render(m.Tmpl, data)
		if body == "" {
			problems = append(problems, m.Name+" is empty")
			continue
		}
		if !all && !strings.Contains(body, "```") {
			skipped++
			continue
		}
		if n := len([]rune(body)); n > discordLimit {
			problems = append(problems, fmt.Sprintf("%s is %d characters, over the %d limit", m.Name, n, discordLimit))
			continue
		}
		collectOdd(body, odd)

		label := "-# " + m.Name + " · " + previewLabel(data)
		if len([]rune(body))+1+len([]rune(label)) <= discordLimit {
			g.preview(channelID, body+"\n"+label)
		} else {
			g.preview(channelID, body)
			g.preview(channelID, label)
		}
		posted++
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Posted %d messages above.", posted)
	if skipped > 0 {
		fmt.Fprintf(&sb, " Skipped %d without banners, use `all:True` to include them.", skipped)
	}
	if len(problems) > 0 {
		fmt.Fprintf(&sb, "\n\nProblems:\n- %s", strings.Join(problems, "\n- "))
	}
	if list := oddList(odd); list != "" {
		fmt.Fprintf(&sb, "\n\nThese sit outside the ranges Discord keeps monospace, so they will drift:\n%s", list)
	}
	return sb.String()
}

func (g *Game) preview(channelID, body string) {
	g.dg.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         body,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	})
	time.Sleep(400 * time.Millisecond)
}

func safeInCodeBlock(r rune) bool {
	switch {
	case r <= unicode.MaxASCII:
		return true
	case r == '🐐':
		return true
	case r >= 0x00A0 && r <= 0x00FF:
		return true
	case r >= 0x2010 && r <= 0x203A:
		return true
	case r >= 0x2500 && r <= 0x259F:
		return true
	case r >= 0x2800 && r <= 0x28FF:
		return true
	}
	return false
}

func collectOdd(body string, into map[rune]bool) {
	for _, r := range body {
		if !safeInCodeBlock(r) {
			into[r] = true
		}
	}
}

func oddList(odd map[rune]bool) string {
	if len(odd) == 0 {
		return ""
	}
	runes := make([]rune, 0, len(odd))
	for r := range odd {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })

	parts := make([]string, 0, len(runes))
	for _, r := range runes {
		parts = append(parts, fmt.Sprintf("`%c` U+%04X", r, r))
	}
	return strings.Join(parts, "  ")
}

func previewData() MsgData {
	deadline := time.Now().Add(time.Duration(5+rand.IntN(55)) * time.Minute)
	stamp := func(t time.Time, style string) string {
		return fmt.Sprintf("<t:%d:%s>", t.Unix(), style)
	}
	sets := []string{"she", "he", "they"}
	loser := "@Alice"
	if rand.IntN(2) == 0 {
		loser = ""
	}
	return MsgData{
		Holder:         "@Alice",
		Winner:         "@Bob",
		Loser:          loser,
		Actor:          "@Mod",
		User:           "@Charlie",
		Challengers:    "@Bob, @Charlie, and @Dave",
		Count:          1 + rand.IntN(4),
		Streak:         rand.IntN(5),
		Deadline:       stamp(deadline, "R"),
		DeadlineAt:     stamp(deadline, "t"),
		HeldSince:      stamp(time.Now().Add(-3*time.Hour), "R"),
		Held:           "3 hours",
		Free:           "22 minutes",
		Total:          rand.IntN(50),
		Escapes:        rand.IntN(5),
		JourneysHome:   rand.IntN(3),
		Minutes:        []int{1, 30, 60}[rand.IntN(3)],
		Channel:        "#goat",
		From:           stamp(time.Now().Add(-4*time.Hour), "t"),
		To:             stamp(time.Now().Add(-3*time.Hour), "t"),
		Lines:          previewLines(stamp),
		Reason:         "Missing Permissions",
		Transient:      rand.IntN(2) == 0,
		Friend:         rand.IntN(2) == 0,
		HolderPronouns: Pronouns{set: sets[rand.IntN(len(sets))]},
		WinnerPronouns: Pronouns{set: sets[rand.IntN(len(sets))]},
	}
}

func previewLines(stamp func(time.Time, string) string) string {
	at := func(h time.Duration) string { return stamp(time.Now().Add(-h*time.Hour), "t") }
	return strings.Join([]string{
		render(HistoryDayMsg, MsgData{From: stamp(time.Now(), "D")}),
		render(HistoryLineMsg, MsgData{User: "@Alice", From: at(4), To: at(3)}),
		render(HistoryLineMsg, MsgData{User: "@Bob", From: at(6), To: at(4)}),
	}, "\n")
}

func previewLabel(d MsgData) string {
	parts := []string{
		"holder " + d.HolderPronouns.They(),
		"winner " + d.WinnerPronouns.They(),
		fmt.Sprintf("count %d", d.Count),
		fmt.Sprintf("streak %d", d.Streak),
		fmt.Sprintf("%d min rounds", d.Minutes),
	}
	if d.Friend {
		parts = append(parts, "friend")
	}
	if d.Loser == "" {
		parts = append(parts, "no previous keeper")
	}
	if d.Transient {
		parts = append(parts, "transient error")
	}
	return strings.Join(parts, " · ")
}
