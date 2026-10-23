package bot

import (
	"errors"
	"log"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/unicode/norm"
)

func CommandDefs() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "goat",
			Description: DescGoat,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "steal",
					Description: DescSteal,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "pet",
					Description: DescPet,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "status",
					Description: DescStatus,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "history",
					Description: DescHistory,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "show",
					Description: DescShow,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		{
			Name:                     "goat-admin",
			Description:              DescGoatAdmin,
			DefaultMemberPermissions: ptr(int64(discordgo.PermissionManageServer)),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "setup",
					Description: DescSetup,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "reset",
					Description: DescReset,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "resolve",
					Description: DescResolve,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "clear-home",
					Description: DescClearHome,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "pronouns",
					Description: DescPronouns,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options:     pronounOptionDefs(),
				},
				{
					Name:        "clear-pronouns",
					Description: DescClearPronouns,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "preview",
					Description: DescPreview,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "all",
							Description: DescPreviewAll,
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Required:    false,
						},
					},
				},
				{
					Name:        "config",
					Description: DescConfig,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "channel",
							Description: DescConfigChannel,
							Type:        discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildText,
							},
							Required: false,
						},
						{
							Name:        "minutes",
							Description: DescConfigMinutes,
							Type:        discordgo.ApplicationCommandOptionInteger,
							MinValue:    ptr(1.0),
							Required:    false,
						},
						{
							Name:        "home",
							Description: DescConfigHome,
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
					},
				},
			},
		},
	}
}

func ptr[T any](v T) *T { return &v }

func (h handler) autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	if data.Name != "goat-admin" || len(data.Options) == 0 || data.Options[0].Name != "pronouns" {
		return
	}
	typed := ""
	for _, o := range data.Options[0].Options {
		if o.Focused {
			typed = o.StringValue()
		}
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: roleSuggestions(h.guildRoles(s, i.GuildID), i.GuildID, typed)},
	})
	if err != nil {
		log.Printf("discord: could not send role suggestions: %v", err)
	}
}

func (h handler) guildRoles(s *discordgo.Session, guildID string) []*discordgo.Role {
	if g, err := s.State.Guild(guildID); err == nil && len(g.Roles) > 0 {
		return g.Roles
	}
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		log.Printf("discord: could not list roles: %v", err)
	}
	return roles
}

func plainText(s string) string {
	return strings.ToLower(norm.NFKC.String(s))
}

func roleSuggestions(roles []*discordgo.Role, guildID, typed string) []*discordgo.ApplicationCommandOptionChoice {
	want := plainText(typed)
	var slashed, rest []*discordgo.Role
	for _, r := range roles {
		if r.ID == guildID || r.Managed {
			continue
		}
		name := plainText(r.Name)
		if !strings.Contains(name, want) {
			continue
		}
		if strings.Contains(name, "/") {
			slashed = append(slashed, r)
		} else {
			rest = append(rest, r)
		}
	}
	highestFirst := func(rs []*discordgo.Role) {
		sort.SliceStable(rs, func(a, b int) bool { return rs[a].Position > rs[b].Position })
	}
	highestFirst(slashed)
	highestFirst(rest)

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, 25)
	for _, r := range append(slashed, rest...) {
		if len(choices) == 25 {
			break
		}
		name := []rune(r.Name)
		if len(name) > 100 {
			name = name[:100]
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: string(name), Value: r.ID})
	}
	return choices
}

func resolveRoleValue(roles []*discordgo.Role, guildID, value string) (string, bool) {
	want := plainText(strings.TrimSpace(value))
	if want == "" {
		return "", false
	}
	var partial []string
	for _, r := range roles {
		if r.ID == guildID || r.Managed {
			continue
		}
		if r.ID == value || plainText(r.Name) == want {
			return r.ID, true
		}
		if strings.Contains(plainText(r.Name), want) {
			partial = append(partial, r.ID)
		}
	}
	if len(partial) == 1 {
		return partial[0], true
	}
	return "", false
}

func pronounOptionDefs() []*discordgo.ApplicationCommandOption {
	descs := map[string]string{
		"she":     DescPronounsShe,
		"he":      DescPronounsHe,
		"they":    DescPronounsThey,
		"heshe":   DescPronounsHeShe,
		"theyshe": DescPronounsTheyShe,
		"theyhe":  DescPronounsTheyHe,
	}
	opts := make([]*discordgo.ApplicationCommandOption, 0, len(pronounOptions))
	for _, name := range pronounOptions {
		opts = append(opts, &discordgo.ApplicationCommandOption{
			Name:         name,
			Description:  descs[name],
			Type:         discordgo.ApplicationCommandOptionString,
			Autocomplete: true,
			Required:     false,
		})
	}
	return opts
}

func (g *Game) Route(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handler{game: g}.route(s, i)
}

type handler struct {
	game *Game
}

func (h handler) route(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID != h.game.cfg.GuildID {
		return
	}
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		h.autocomplete(s, i)
		return
	}
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	sub := ""
	if len(data.Options) > 0 && data.Options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		sub = data.Options[0].Name
	}

	public := data.Name == "goat" && (sub == "status" || sub == "history" || ((sub == "show" || sub == "pet") && h.game.IsHolder(callerID(i))))
	if err := defer_(s, i, public); err != nil {
		log.Printf("discord: could not acknowledge interaction: %v", err)
		return
	}

	var body string
	switch data.Name {
	case "goat":
		body = h.goat(i, sub)
	case "goat-admin":
		body = h.admin(s, i, sub, data)
	}

	if strings.TrimSpace(body) == "" {
		body = EmptySlotNotice
	}
	if strings.TrimSpace(body) == "" {
		body = "."
	}
	edit(s, i, body)
}

func (h handler) steal(i *discordgo.InteractionCreate) string {
	data, err := h.game.Steal(callerID(i))
	switch {
	case errors.Is(err, ErrNoRound):
		return render(NoRoundMsg, data)
	case errors.Is(err, ErrIsHolder):
		return render(YouHoldItMsg, data)
	case errors.Is(err, ErrAlreadyEntry):
		return render(AlreadyEnteredMsg, data)
	default:
		return render(EnteredMsg, data)
	}
}

func (h handler) goat(i *discordgo.InteractionCreate, sub string) string {
	switch sub {
	case "steal":
		return h.steal(i)
	case "pet":
		return h.pet(i)
	case "status":
		return h.status()
	case "history":
		return h.history()
	case "show":
		return h.show(i)
	}
	return ""
}

func (h handler) admin(s *discordgo.Session, i *discordgo.InteractionCreate, sub string, data discordgo.ApplicationCommandInteractionData) string {
	if !isAdmin(i) {
		return render(NotAdminMsg, MsgData{})
	}

	opts := subOptions(data)
	switch sub {
	case "setup":
		out, err := h.game.Setup(i.ChannelID)
		switch {
		case errors.Is(err, ErrRunning):
			return render(AlreadyRunningMsg, out)
		case err != nil:
			return render(RoleMissingMsg, out)
		}
		return render(ClaimedMsg, out)
	case "reset":
		out, err := h.game.Reset(callerID(i))
		if err != nil {
			return render(RoleMissingMsg, out)
		}
		return render(ResetMsg, out)
	case "resolve":
		if err := h.game.ResolveNow(); errors.Is(err, ErrNoRound) {
			return render(NoRoundMsg, MsgData{})
		}
		return h.status()
	case "clear-home":
		return render(ConfigDoneMsg, h.game.ClearHome())
	case "pronouns":
		guildRoles := h.guildRoles(s, i.GuildID)
		chosen := map[string]string{}
		var unknown []string
		for _, name := range pronounOptions {
			o := option(opts, name)
			if o == nil {
				continue
			}
			value := o.StringValue()
			if roleID, ok := resolveRoleValue(guildRoles, i.GuildID, value); ok {
				chosen[name] = roleID
			} else {
				unknown = append(unknown, "- "+name+": "+value)
			}
		}
		if len(unknown) > 0 {
			return render(PronounsUnknownMsg, MsgData{Lines: strings.Join(unknown, "\n")})
		}
		mapped := h.game.ConfigPronouns(chosen)
		var lines []string
		for _, name := range pronounOptions {
			if roleID := mapped[name]; roleID != "" {
				lines = append(lines, "- "+name+": <@&"+roleID+">")
			}
		}
		return render(PronounsSetMsg, MsgData{Lines: strings.Join(lines, "\n")})
	case "clear-pronouns":
		return render(ConfigDoneMsg, h.game.ClearPronouns())
	case "preview":
		all := false
		if o := option(opts, "all"); o != nil {
			all = o.BoolValue()
		}
		return h.game.Preview(i.ChannelID, all)
	case "config":
		channelID := ""
		if o := option(opts, "channel"); o != nil {
			channelID = o.ChannelValue(nil).ID
		}
		minutes := 0
		if o := option(opts, "minutes"); o != nil {
			minutes = int(o.IntValue())
		}
		homeID := ""
		if o := option(opts, "home"); o != nil {
			homeID = o.UserValue(nil).ID
		}
		out, err := h.game.Config(channelID, homeID, minutes)
		if errors.Is(err, ErrNoChannel) {
			return render(ChannelMissingMsg, out)
		}
		return render(ConfigDoneMsg, out)
	}
	return ""
}

func (h handler) pet(i *discordgo.InteractionCreate) string {
	data, out, err := h.game.Pet(callerID(i))
	switch {
	case errors.Is(err, ErrNoRound):
		return render(NoRoundMsg, data)
	case errors.Is(err, ErrNotHolder):
		return render(ShowNotHolderMsg, data)
	case errors.Is(err, ErrAlreadyPetted):
		return render(AlreadyPettedMsg, data)
	}
	switch out {
	case petBrokeFree:
		return render(PetBrokeFreeMsg, data)
	case petBonusRound:
		return render(PetBonusMsg, data)
	default:
		return render(PetNoEffectMsg, data)
	}
}

func (h handler) status() string {
	started, unheld, data := h.game.StatusData()
	switch {
	case !started:
		return render(NoRoundMsg, data)
	case unheld:
		return render(StatusUnheldMsg, data)
	default:
		return render(StatusMsg, data)
	}
}

func (h handler) show(i *discordgo.InteractionCreate) string {
	started, unheld, data := h.game.StatusData()
	switch {
	case !started:
		return render(NoRoundMsg, data)
	case unheld:
		return render(StatusUnheldMsg, data)
	case data.Holder == mention(callerID(i)):
		return render(ShowGoatMsg, data)
	default:
		return render(ShowNotHolderMsg, data)
	}
}

func (h handler) history() string {
	reigns, total, escapes, journeys := h.game.HistoryData()
	if len(reigns) == 0 {
		return render(HistoryEmptyMsg, MsgData{Total: total, Escapes: escapes, JourneysHome: journeys})
	}

	lines := make([]string, 0, len(reigns))
	day := ""
	for n := len(reigns) - 1; n >= 0; n-- {
		r := reigns[n]
		if d := r.From.Format("2006-01-02"); d != day {
			day = d
			if heading := render(HistoryDayMsg, MsgData{From: dayStamp(r.From)}); heading != "" {
				if len(lines) > 0 {
					lines = append(lines, "")
				}
				lines = append(lines, heading)
			}
		}
		line := render(HistoryLineMsg, MsgData{
			User:   mention(r.UserID),
			From:   clockTime(r.From),
			To:     clockTime(r.To),
			Streak: r.Streak,
		})
		if line != "" {
			lines = append(lines, line)
		}
	}
	return render(HistoryHeaderMsg, MsgData{Lines: strings.Join(lines, "\n"), Total: total, Escapes: escapes, JourneysHome: journeys})
}

func defer_(s *discordgo.Session, i *discordgo.InteractionCreate, public bool) error {
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{},
	}
	if !public {
		resp.Data.Flags = discordgo.MessageFlagsEphemeral
	}
	return s.InteractionRespond(i.Interaction, resp)
}

func edit(s *discordgo.Session, i *discordgo.InteractionCreate, body string) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content:         &body,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers}},
	})
	if err != nil {
		log.Printf("discord: could not send reply: %v", err)
	}
}

func callerID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func isAdmin(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	const mask = discordgo.PermissionManageServer | discordgo.PermissionAdministrator
	return i.Member.Permissions&mask != 0
}

func subOptions(data discordgo.ApplicationCommandInteractionData) []*discordgo.ApplicationCommandInteractionDataOption {
	if len(data.Options) == 0 {
		return nil
	}
	return data.Options[0].Options
}

func option(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range opts {
		if o.Name == name {
			return o
		}
	}
	return nil
}
