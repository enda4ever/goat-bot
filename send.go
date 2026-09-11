package main

import (
	"log"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
)

func render(tmpl *template.Template, data MsgData) string {
	if tmpl == nil {
		return ""
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		log.Printf("strings: template %q failed: %v", tmpl.Name(), err)
		return ""
	}
	return strings.TrimSpace(sb.String())
}

func (g *Game) announce(tmpl *template.Template, data MsgData) {
	body := render(tmpl, data)
	if body == "" {
		return
	}
	if g.s.AnnounceChannel == "" {
		log.Printf("discord: no announcement channel set, dropping %q", tmpl.Name())
		return
	}
	_, err := g.dg.ChannelMessageSendComplex(g.s.AnnounceChannel, &discordgo.MessageSend{
		Content:         body,
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers}},
	})
	if err != nil {
		log.Printf("discord: could not post to channel %s: %v", g.s.AnnounceChannel, err)
	}
}

func relTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return "<t:" + strconv.FormatInt(t.Unix(), 10) + ":R>"
}

func clockTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return "<t:" + strconv.FormatInt(t.Unix(), 10) + ":t>"
}

func channelRef(channelID string) string {
	if channelID == "" {
		return ""
	}
	return "<#" + channelID + ">"
}

func mentionList(userIDs []string) string {
	parts := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		parts = append(parts, mention(id))
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + ChallengerLastJoin + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ChallengerJoin) + ChallengerLastJoin + parts[len(parts)-1]
	}
}
