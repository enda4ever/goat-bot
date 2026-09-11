package main

import "text/template"

// Every user-facing string the bot can produce lives in this file, and nothing
// else in the codebase contains player-visible text.
//
// HOW TO FILL THIS IN
//
//   - Write your text between the backticks. Multi-line is fine.
//   - A slot left empty is never sent. Nothing is ever substituted for you, so
//     an unfinished file makes a quiet bot, not a bot leaking placeholders.
//   - {{.Field}} inserts a value. Each slot lists the fields it populates; a
//     field not listed for that slot will render as empty.
//   - ASCII art must go inside a ``` fence or Discord's proportional font will
//     misalign it. A message caps at 2000 characters, art included.
//   - To write a literal backtick inside a raw string, close the raw string and
//     concatenate: `text ` + "`" + ` more text`.
//
// TIME FIELDS
//
// Times arrive as Discord timestamp markup, which each reader sees in their own
// timezone and which counts down live without the bot editing anything:
//
//	{{.Deadline}}     -> "in 58 minutes"     (relative, updates by itself)
//	{{.DeadlineAt}}   -> "3:00 PM"           (that reader's local clock time)
//	{{.HeldSince}}    -> "3 hours ago"       (relative)
//
// MENTIONS
//
// Mention fields render as real pings. Only user mentions are permitted; the
// bot cannot ping a role or everyone even if a template asks it to.

// MsgData is the set of values available to the templates below. Any field may
// be referenced from any slot, but only the ones documented on a slot are
// filled in for it.
type MsgData struct {
	Holder     string
	Winner     string
	Loser      string
	Actor      string
	Challengers string
	Count      int
	Streak     int
	Deadline   string
	DeadlineAt string
	HeldSince  string
	Total      int
	Minutes    int
	Channel    string
	Lines      string
	User       string
	From       string
	To         string
	Reason     string
}

func msg(name, body string) *template.Template {
	return template.Must(template.New(name).Parse(body))
}

// How {{.Challengers}} glues the entrant mentions together. With three
// entrants and the values below you get "@a, @b, and @c". Set both to ", " for
// a flat list, or "\n" to put one per line.
var (
	ChallengerJoin     = ", "
	ChallengerLastJoin = ", and "
)

// ---------------------------------------------------------------------------
// PUBLIC ANNOUNCEMENTS  (posted in the announcement channel, visible to all)
// ---------------------------------------------------------------------------

// Sent once, when an admin runs /goat setup and the game begins.
// Fields: .Holder .Deadline .DeadlineAt .Minutes
var ClaimedMsg = msg("claimed", ``)

// Sent at the end of a round in which nobody entered. The holder keeps it.
// Fields: .Holder .Streak .Deadline .DeadlineAt .HeldSince
var SurvivedMsg = msg("survived", ``)

// Sent at the end of a round that one or more members entered. .Winner was
// drawn from .Challengers and now holds the goat; .Loser held it before.
// Fields: .Winner .Loser .Challengers .Count .Deadline .DeadlineAt .Total
var StolenMsg = msg("stolen", ``)

// Sent when the holder left the server and an entrant picked the goat up. This
// is a recovery, not a theft: .Winner never beat .Loser, who simply vanished.
// Fields: .Winner .Loser .Challengers .Count .Deadline .DeadlineAt .Total
var EscapedRecoveredMsg = msg("escaped_recovered", ``)

// Sent when the holder left the server and nobody had entered, so the goat is
// now unheld and waiting for the next round's entrants.
// Fields: .Loser .Deadline .DeadlineAt
var EscapedUnheldMsg = msg("escaped_unheld", ``)

// Sent when an admin runs /goat reset and the game restarts on a new holder.
// Fields: .Holder .Actor .Deadline .DeadlineAt
var ResetMsg = msg("reset", ``)

// ---------------------------------------------------------------------------
// PRIVATE REPLIES  (ephemeral: only the member who ran the command sees these)
// ---------------------------------------------------------------------------

// /steal succeeded. They are in the draw for this round.
// Fields: .Count .Deadline .DeadlineAt
var EnteredMsg = msg("entered", ``)

// /steal when they already entered this round. One entry each, per the rules.
// Fields: .Count .Deadline .DeadlineAt
var AlreadyEnteredMsg = msg("already_entered", ``)

// /steal by the current holder. They cannot steal from themselves.
// Fields: .Streak .Deadline .DeadlineAt
var YouHoldItMsg = msg("you_hold_it", ``)

// /steal before /goat setup has ever been run, so there is no round yet.
// Fields: none
var NoRoundMsg = msg("no_round", ``)

// An admin-only subcommand was run by somebody without Manage Server.
// Fields: none
var NotAdminMsg = msg("not_admin", ``)

// /goat setup when a game is already running. Refuses, so a stray setup can
// never wipe an ongoing game. Points at /goat reset instead.
// Fields: .Holder
var AlreadyRunningMsg = msg("already_running", ``)

// /goat config confirmation.
// Fields: .Channel .Minutes
var ConfigDoneMsg = msg("config_done", ``)

// ---------------------------------------------------------------------------
// READOUTS  (public replies to /goat status and /goat history)
// ---------------------------------------------------------------------------

// /goat status while somebody holds the goat.
// Fields: .Holder .Streak .Count .Deadline .DeadlineAt .HeldSince .Total
var StatusMsg = msg("status", ``)

// /goat status while the goat is unheld, because the holder left the server.
// Fields: .Count .Deadline .DeadlineAt
var StatusUnheldMsg = msg("status_unheld", ``)

// /goat history. .Lines is every HistoryLineMsg below, newest first, joined
// with newlines.
// Fields: .Lines .Total
var HistoryHeaderMsg = msg("history_header", ``)

// One row inside HistoryHeaderMsg's .Lines. Rendered once per past reign.
// Keep it to a single line.
// Fields: .User .From .To .Streak
var HistoryLineMsg = msg("history_line", ``)

// /goat history before anybody has lost the goat.
// Fields: none
var HistoryEmptyMsg = msg("history_empty", ``)

// ---------------------------------------------------------------------------
// WARNINGS  (things an admin needs to fix)
// ---------------------------------------------------------------------------

// Posted publicly instead of a theft announcement when the goat role could not
// be moved, which almost always means the bot's own role sits below the goat
// role and needs dragging above it. No transfer happened: the previous holder
// keeps the goat, so this must not congratulate anyone.
// Fields: .Winner .Holder .Reason
var RoleFailedMsg = msg("role_failed", ``)

// Ephemeral. An admin command ran but no announcement channel is configured.
// Fields: none
var ChannelMissingMsg = msg("channel_missing", ``)

// Ephemeral. The goat role was deleted out from under the bot and could not be
// recreated.
// Fields: .Reason
var RoleMissingMsg = msg("role_missing", ``)

// ---------------------------------------------------------------------------
// COMMAND PICKER TEXT
// ---------------------------------------------------------------------------
//
// What members read in Discord's slash-command autocomplete. Descriptions are
// capped at 100 characters. The command and option NAMES are not here: Discord
// requires those to be lowercase with no spaces, and changing one would change
// what players type, so they are fixed in commands.go.

var (
	DescSteal         = "Enter this hour's draw to take the goat"
	DescGoat          = "The goat"
	DescStatus        = "Who has the goat and how long is left"
	DescHistory       = "Recent goat holders"
	DescSetup         = "Start the game (admin)"
	DescSetupMember   = "Who starts with the goat (default: you)"
	DescReset         = "Restart the game on a new holder (admin)"
	DescResetMember   = "Who gets the goat (default: you)"
	DescResolve       = "End the current round right now (admin)"
	DescConfig        = "Set the announcement channel and round length (admin)"
	DescConfigChannel = "Where public announcements go"
	DescConfigMinutes = "Round length in minutes, applied from the next round"
)

// Shown when a command's reply slot above is still empty. Discord requires
// every command to get some reply, so this cannot be blank; it exists to make
// an unwritten slot obvious rather than to be read by players.
var EmptySlotNotice = "(no text written for this message yet)"
