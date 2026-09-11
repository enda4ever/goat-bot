package main

import (
	"errors"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type fakeDiscord struct {
	present map[string]bool
	roles   map[string][]string
	sent    []string

	failRoleAdd    error
	failRoleRemove error

	addCalls    int
	removeCalls int
}

func newFake(members ...string) *fakeDiscord {
	f := &fakeDiscord{
		present: map[string]bool{},
		roles:   map[string][]string{},
	}
	for _, m := range members {
		f.present[m] = true
	}
	return f
}

func notFound() error {
	return &discordgo.RESTError{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
}

func (f *fakeDiscord) leave(userID string) {
	delete(f.present, userID)
}

func (f *fakeDiscord) holdersOf(roleID string) []string {
	var out []string
	for user, roles := range f.roles {
		for _, r := range roles {
			if r == roleID {
				out = append(out, user)
			}
		}
	}
	return out
}

func (f *fakeDiscord) GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
	if !f.present[userID] {
		return nil, notFound()
	}
	return &discordgo.Member{User: &discordgo.User{ID: userID}}, nil
}

func (f *fakeDiscord) GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	f.addCalls++
	if f.failRoleAdd != nil {
		return f.failRoleAdd
	}
	for _, r := range f.roles[userID] {
		if r == roleID {
			return nil
		}
	}
	f.roles[userID] = append(f.roles[userID], roleID)
	return nil
}

func (f *fakeDiscord) GuildMemberRoleRemove(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	f.removeCalls++
	if f.failRoleRemove != nil {
		return f.failRoleRemove
	}
	kept := f.roles[userID][:0]
	for _, r := range f.roles[userID] {
		if r != roleID {
			kept = append(kept, r)
		}
	}
	f.roles[userID] = kept
	return nil
}

func (f *fakeDiscord) GuildRoles(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Role, error) {
	return []*discordgo.Role{{ID: "role-goat", Name: goatRoleName}}, nil
}

func (f *fakeDiscord) GuildRoleCreate(guildID string, data *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error) {
	return &discordgo.Role{ID: "role-new", Name: goatRoleName}, nil
}

func (f *fakeDiscord) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.sent = append(f.sent, data.Content)
	return &discordgo.Message{ID: "m"}, nil
}

type memStore struct {
	saved *State
	saves int
}

func (m *memStore) Load() (*State, error) {
	if m.saved == nil {
		return &State{RoundMinutes: defaultRoundMinutes}, nil
	}
	return m.saved, nil
}

func (m *memStore) Save(s *State) error {
	m.saves++
	m.saved = s
	return nil
}

var errForced = errors.New("forced failure")
