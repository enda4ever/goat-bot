package bot

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
	Holder      string
	Winner      string
	Loser       string
	Actor       string
	Challengers string
	Count       int
	Streak      int
	Deadline    string
	DeadlineAt  string
	HeldSince   string
	Total       int
	Minutes     int
	Channel     string
	Lines       string
	User        string
	From        string
	To          string
	Reason      string
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
var ClaimedMsg = msg("claimed", "```"+`
  ────────────────────── ·𖤍· ─────────────────────
  ! ! !   𝕥 𝕙 𝕖   𝕘 𝕠 𝕒 𝕥    𝕚 𝕤    𝕝 𝕠 𝕠 𝕤 𝕖   ! ! !
 ༺──────────────────────────────────────────────༻
`+"```"+`
{{.Holder}} holds the goat.
The next holder is announced {{.Deadline}}.`)

// Sent at the end of a round in which nobody entered. The holder keeps it.
// Fields: .Holder .Streak .Deadline .DeadlineAt .HeldSince
var SurvivedMsg = msg("survived", "```"+`
  ────────────────────── ·𖤍· ─────────────────────
  . . .   𝕥 𝕙 𝕖   𝕘 𝕠 𝕒 𝕥   𝕤 𝕥 𝕒 𝕪 𝕤   𝕡 𝕦 𝕥   . . .
 ༺──────────────────────────────────────────────༻
`+"```"+`
Not one soul reached for it. {{.Holder}} keeps the goat,
unchallenged {{.Streak}} rounds running,
held since {{.HeldSince}}.
The next thieving is settled {{.Deadline}}.`)

// Sent at the end of a round that one or more members entered. .Winner was
// drawn from .Challengers and now holds the goat; .Loser held it before.
// Fields: .Winner .Loser .Challengers .Count .Deadline .DeadlineAt .Total
var StolenMsg = msg("stolen", "```"+`
  ────────────────────── ·𖤍· ─────────────────────
  ! ! !   𝕥 𝕙 𝕖   𝕘 𝕠 𝕒 𝕥   𝕚 𝕤   𝕤 𝕥 𝕠 𝕝 𝕖 𝕟   ! ! !
 ༺──────────────────────────────────────────────༻
`+"```"+`
{{.Winner}} has taken the goat from {{.Loser}}.
{{.Count}} reached for it: {{.Challengers}}.
Only one closed a hand around it.
That makes {{.Total}} times the goat has changed keeper.
The next thieving is settled {{.Deadline}}.`)

// Sent when the holder left the server and an entrant picked the goat up. This
// is a recovery, not a theft: .Winner never beat .Loser, who simply vanished.
// Fields: .Winner .Loser .Challengers .Count .Deadline .DeadlineAt .Total
var EscapedRecoveredMsg = msg("escaped_recovered", "```"+`
  ───────────────────────── ·𖤍· ────────────────────────
  ! ! !   𝕥 𝕙 𝕖   𝕘 𝕠 𝕒 𝕥   𝕚 𝕤   𝕣 𝕖 𝕔 𝕠 𝕧 𝕖 𝕣 𝕖 𝕕   ! ! !
 ༺────────────────────────────────────────────────────༻
`+"```"+`
{{.Loser}} fled the server with the goat under one arm.
No theft, then, and no glory in it.
{{.Winner}} found the creature wandering and led it home,
out of {{.Count}} who came looking: {{.Challengers}}.
The next thieving is settled {{.Deadline}}.`)

// Sent when the holder left the server and nobody had entered, so the goat is
// now unheld and waiting for the next round's entrants.
// Fields: .Loser .Deadline .DeadlineAt
var EscapedUnheldMsg = msg("escaped_unheld", "```"+`
  ──────────────────────── ·𖤍· ────────────────────────
 𓂃⠀˖⠀𓇬  𝕥 𝕙 𝕖   𝕘 𝕠 𝕒 𝕥   𝕨 𝕒 𝕟 𝕕 𝕖 𝕣 𝕤   𝕗 𝕣 𝕖 𝕖  𓇬⠀˖⠀𓂃
 ༺───────────────────────────────────────────────────༻
`+"```"+`
{{.Loser}} fled the server and the goat went with them.
Nobody was reaching for it, so it belongs to no one at all.
It falls to whoever swipes it first, {{.Deadline}}.`)

// Sent when an admin runs /goat reset and the game restarts on a new holder.
// Fields: .Holder .Actor .Deadline .DeadlineAt
var ResetMsg = msg("reset", "```"+`
  ──────────────────── ·𖤍· ──────────────────
   𓂃⠀˖⠀𓇬    𝕒   𝕟 𝕖 𝕨   𝕠 𝕣 𝕕 𝕖 𝕣    𓇬⠀˖⠀𓂃⠀ 
 ༺─────────────────────────────────────────༻
`+"```"+`
{{.Actor}} has torn up the old order.
{{.Holder}} holds the goat now.
The next thieving is settled {{.Deadline}}.`)

// ---------------------------------------------------------------------------
// PRIVATE REPLIES  (ephemeral: only the member who ran the command sees these)
// ---------------------------------------------------------------------------

// /steal succeeded. They are in the draw for this round.
// Fields: .Count .Deadline .DeadlineAt
var EnteredMsg = msg("entered", `You have attempted to steal the goat! Within the hour, all shall be revealed`)

// /steal when they already entered this round. One entry each, per the rules.
// Fields: .Count .Deadline .DeadlineAt
var AlreadyEnteredMsg = msg("already_entered", `You have already attempted to steal the goat this hour! Be patient.`)

// /steal by the current holder. They cannot steal from themselves.
// Fields: .Streak .Deadline .DeadlineAt
var YouHoldItMsg = msg("you_hold_it", `The goat is already yours! One cannot steal from oneself. Hold fast, and learn {{.Deadline}} whether anybody swiped it from you.`)

// /steal before /goat setup has ever been run, so there is no round yet.
// Fields: none
var NoRoundMsg = msg("no_round", `Currently NO ONE has the goat. Be patient.`)

// An admin-only subcommand was run by somebody without Manage Server.
// Fields: none
var NotAdminMsg = msg("not_admin", `Forbidden attempt to command the goat's fate!`)

// /goat setup when a game is already running. Refuses, so a stray setup can
// never wipe an ongoing game. Points at /goat reset instead.
// Fields: .Holder
var AlreadyRunningMsg = msg("already_running", `The goat is already among the peasants. No setup required.`)

// /goat config confirmation.
// Fields: .Channel .Minutes
var ConfigDoneMsg = msg("config_done", `A successful command!`)

// ---------------------------------------------------------------------------
// READOUTS  (public replies to /goat status and /goat history)
// ---------------------------------------------------------------------------

// /goat status while somebody holds the goat.
// Fields: .Holder .Streak .Count .Deadline .DeadlineAt .HeldSince .Total
var StatusMsg = msg("status", `The current holder is {{.Holder}}. The next stealing results arrive at {{.DeadlineAt}}`)

// /goat status while the goat is unheld, because the holder left the server.
// Fields: .Count .Deadline .DeadlineAt
var StatusUnheldMsg = msg("status_unheld", `The goat is roaming free! Attempt to steal before {{.DeadlineAt}}`)

// /goat history. .Lines is every HistoryLineMsg below, newest first, joined
// with newlines.
// Fields: .Lines .Total
var HistoryHeaderMsg = msg("history_header", "```"+`
  ────────────────── ·𖤍· ──────────────────
 𓂃⠀˖⠀𓇬    𝕥 𝕙 𝕖  𝕘 𝕠 𝕒 𝕥  𝕝 𝕖 𝕕 𝕘 𝕖 𝕣    𓇬⠀˖⠀𓂃
 ༺───────────────────────────────────────༻
`+"```"+`
{{.Total}} keepers have come and gone.
{{.Lines}}`)

// One row inside HistoryHeaderMsg's .Lines. Rendered once per past reign.
// Keep it to a single line.
// Fields: .User .From .To .Streak
var HistoryLineMsg = msg("history_line", `{{.User}} held it from {{.From}} until {{.To}}, unchallenged {{.Streak}} rounds of it.`)

// /goat history before anybody has lost the goat.
// Fields: none
var HistoryEmptyMsg = msg("history_empty", `The goat has never changed hands. The ledger begins with the first successful theft.`)

// ---------------------------------------------------------------------------
// WARNINGS  (things an admin needs to fix)
// ---------------------------------------------------------------------------

// Posted publicly instead of a theft announcement when the goat role could not
// be moved, which almost always means the bot's own role sits below the goat
// role and needs dragging above it. No transfer happened: the previous holder
// keeps the goat, so this must not congratulate anyone.
// Fields: .Winner .Holder .Reason
var RoleFailedMsg = msg("role_failed", "```"+`
  ────────────────────────── ·𖤍· ─────────────────────────
  . . .   𝕥 𝕙 𝕖   𝕘 𝕠 𝕒 𝕥   𝕨 𝕚 𝕝 𝕝   𝕟 𝕠 𝕥   𝕞 𝕠 𝕧 𝕖   . . .
 ༺──────────────────────────────────────────────────────༻
`+"```"+`
The goat could not be handed over, so {{.Holder}} keeps it and nobody has stolen anything.
Discord said: {{.Reason}}
An admin must drag the bot's own role above the goat role in Server Settings, then run /goat resolve.`)

// Ephemeral. An admin command ran but no announcement channel is configured.
// Fields: none
var ChannelMissingMsg = msg("channel_missing", `Nowhere to announce it. Run /goat config channel:#somewhere first, then try again.`)

// Ephemeral. The goat role was deleted out from under the bot and could not be
// recreated.
// Fields: .Reason
var RoleMissingMsg = msg("role_missing", `The goat role is gone and could not be built again: {{.Reason}}`)

// ---------------------------------------------------------------------------
// COMMAND PICKER TEXT
// ---------------------------------------------------------------------------
//
// What members read in Discord's slash-command autocomplete. Descriptions are
// capped at 100 characters. The command and option NAMES are not here: Discord
// requires those to be lowercase with no spaces, and changing one would change
// what players type, so they are fixed in commands.go.

var (
	DescSteal         = "Creep up on the goat and try to steal it this hour"
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
