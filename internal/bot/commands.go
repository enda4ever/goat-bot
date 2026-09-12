package bot

import (
	"errors"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func CommandDefs() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "steal",
			Description: DescSteal,
		},
		{
			Name:        "goat",
			Description: DescGoat,
			Options: []*discordgo.ApplicationCommandOption{
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
					Name:        "setup",
					Description: DescSetup,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "member",
							Description: DescSetupMember,
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
					},
				},
				{
					Name:        "reset",
					Description: DescReset,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "member",
							Description: DescResetMember,
							Type:        discordgo.ApplicationCommandOptionUser,
							Required:    false,
						},
					},
				},
				{
					Name:        "resolve",
					Description: DescResolve,
					Type:        discordgo.ApplicationCommandOptionSubCommand,
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
					},
				},
			},
		},
	}
}

func ptr[T any](v T) *T { return &v }

func (g *Game) Route(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handler{game: g}.route(s, i)
}

type handler struct {
	game *Game
}

func (h handler) route(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	if i.GuildID != h.game.cfg.GuildID {
		return
	}

	data := i.ApplicationCommandData()
	sub := ""
	if len(data.Options) > 0 && data.Options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		sub = data.Options[0].Name
	}

	public := data.Name == "goat" && (sub == "status" || sub == "history")
	if err := defer_(s, i, public); err != nil {
		log.Printf("discord: could not acknowledge interaction: %v", err)
		return
	}

	var body string
	switch {
	case data.Name == "steal":
		body = h.steal(i)
	case data.Name == "goat":
		body = h.goat(i, sub, data)
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
	case err != nil:
		return render(RoleMissingMsg, MsgData{Reason: err.Error()})
	default:
		return render(EnteredMsg, data)
	}
}

func (h handler) goat(i *discordgo.InteractionCreate, sub string, data discordgo.ApplicationCommandInteractionData) string {
	switch sub {
	case "status":
		return h.status()
	case "history":
		return h.history()
	}

	if !isAdmin(i) {
		return render(NotAdminMsg, MsgData{})
	}

	opts := subOptions(data)
	switch sub {
	case "setup":
		target := userOption(i, opts, "member", callerID(i))
		out, err := h.game.Setup(target, i.ChannelID)
		switch {
		case errors.Is(err, ErrRunning):
			return render(AlreadyRunningMsg, out)
		case err != nil:
			return render(RoleMissingMsg, out)
		}
		return render(ClaimedMsg, out)
	case "reset":
		target := userOption(i, opts, "member", callerID(i))
		out, err := h.game.Reset(target, callerID(i))
		if err != nil {
			return render(RoleMissingMsg, out)
		}
		return render(ResetMsg, out)
	case "resolve":
		if err := h.game.ResolveNow(); errors.Is(err, ErrNoRound) {
			return render(NoRoundMsg, MsgData{})
		}
		return h.status()
	case "config":
		channelID := ""
		if o := option(opts, "channel"); o != nil {
			channelID = o.ChannelValue(nil).ID
		}
		minutes := 0
		if o := option(opts, "minutes"); o != nil {
			minutes = int(o.IntValue())
		}
		out, err := h.game.Config(channelID, minutes)
		if errors.Is(err, ErrNoChannel) {
			return render(ChannelMissingMsg, out)
		}
		return render(ConfigDoneMsg, out)
	}
	return ""
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

func (h handler) history() string {
	reigns, total := h.game.HistoryData()
	if len(reigns) == 0 {
		return render(HistoryEmptyMsg, MsgData{Total: total})
	}

	lines := make([]string, 0, len(reigns))
	for n := len(reigns) - 1; n >= 0; n-- {
		r := reigns[n]
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
	return render(HistoryHeaderMsg, MsgData{Lines: strings.Join(lines, "\n"), Total: total})
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

func userOption(i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption, name, fallback string) string {
	o := option(opts, name)
	if o == nil {
		return fallback
	}
	if u := o.UserValue(nil); u != nil && u.ID != "" {
		return u.ID
	}
	return fallback
}
