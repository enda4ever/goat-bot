package bot

import (
	"errors"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type Discord interface {
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
	GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	GuildMemberRoleRemove(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	GuildRoles(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Role, error)
	GuildRoleCreate(guildID string, data *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

func isNotFound(err error) bool {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Response != nil {
		return rest.Response.StatusCode == http.StatusNotFound
	}
	return false
}

func mention(userID string) string {
	if userID == "" {
		return ""
	}
	return "<@" + userID + ">"
}
