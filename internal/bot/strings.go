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
//	{{.From}}/{{.To}}  -> "1:00 PM"           (ledger rows)
//	{{.HeldSince}}    -> "3 hours ago"       (relative)
//	{{.Held}}         -> "3 hours"           (how long they have held it)
//	{{.Free}}         -> "3 hours"           (how long the goat has been unheld)
//
// MENTIONS
//
// Mention fields render as real pings. Only user mentions are permitted; the
// bot cannot ping a role or everyone even if a template asks it to.

// MsgData is the set of values available to the templates below. Any field may
// be referenced from any slot, but only the ones documented on a slot are
// filled in for it.
type MsgData struct {
	Holder       string
	Winner       string
	Loser        string
	Actor        string
	Challengers  string
	Count        int
	Streak       int
	Deadline     string
	DeadlineAt   string
	HeldSince    string
	Held         string
	Free         string
	Total        int
	Escapes      int
	JourneysHome int
	Minutes      int
	Channel      string
	Lines        string
	User         string
	From         string
	To           string
	Reason       string
	Transient    bool
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

// Sent once, when an admin runs /goat setup and the game begins. The goat
// belongs to nobody yet.
// Fields: .Minutes .Free .Deadline .DeadlineAt
var ClaimedMsg = msg("claimed", "```"+`
  ──────── ·𖤍· ────────
     𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕚𝕤 𝕝𝕠𝕠𝕤𝕖
  ༺───────────────────༻
`+"```"+`
The goat has been set loose upon the Kingdom. Perhaps it will find its rightful home. Or perhaps it will be stolen!
Every {{.Minutes}} {{if eq .Minutes 1}}minute{{else}}minutes{{end}}, the goat becomes restless and unruly. If you attempted a `+"`"+`/goat steal`+"`"+` action before then, you may be declared as the reigning goat keeper! At {{.DeadlineAt}} we shall see who was the most cunning, who was clumsy, and who was gracious.`)

// Sent at the end of a round in which nobody entered. The holder keeps it.
// Fields: .Holder .Streak .Deadline .DeadlineAt .HeldSince .Held
var SurvivedMsg = msg("survived", "```"+`
  ──────── ·𖤍· ────────
     𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕤𝕥𝕒𝕪𝕤 𝕡𝕦𝕥
  ༺───────────────────༻
`+"```"+`
Still, no one is brave enough to attempt another thieving of the goat! {{.Holder}} has held the beast for {{.Held}} now.
{{if gt .Streak 1}}That is {{.Streak}} rounds unchallenged. The Kingdom grows complacent.
{{end}}But the goat will grow restless {{.Deadline}}.`)

// Sent at the end of a round that one or more members entered. .Winner was
// drawn from .Challengers and now holds the goat; .Loser held it before.
// Fields: .Winner .Loser .Challengers .Count .Deadline .DeadlineAt .Total
var StolenMsg = msg("stolen", "```"+`
  ──────── ·𖤍· ────────
     𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕚𝕤 𝕤𝕥𝕠𝕝𝕖𝕟
  ༺───────────────────༻
`+"```"+`
{{if .Loser}}{{.Winner}} has snatched the goat from {{.Loser}}!{{else}}{{.Winner}} has captured the wandering goat before it could find its home!{{end}}
{{if gt .Count 1}}{{.Count}} thieves attempted to snatch it, but only one was cunning enough to succeed.
{{end}}{{.Winner}} may want to sleep with one eye open, as the goat will grow restless {{.Deadline}}.`)

// Sent when the holder left the server and an entrant picked the goat up. This
// is a recovery, not a theft: .Winner never beat .Loser, who simply vanished.
// Fields: .Winner .Loser .Challengers .Count .Deadline .DeadlineAt .Total
var EscapedRecoveredMsg = msg("escaped_recovered", "```"+`
  ──────── ·𖤍· ────────
   𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕚𝕤 𝕣𝕖𝕔𝕒𝕡𝕥𝕦𝕣𝕖𝕕
  ༺───────────────────༻
`+"```"+`
{{.Winner}} has captured the goat! It wanders free no longer. {{.Count}} {{if eq .Count 1}}thief{{else}}thieves{{end}} tried to snatch the beast away from its path home.
Mayhaps they will attempt again, before {{.DeadlineAt}}.`)

// Sent when the draw chose nobody: the goat slipped every hand that reached for
// it and now belongs to no one. .Loser held it before, .Challengers all missed.
// Fields: .Loser .Challengers .Count .Free .Deadline .DeadlineAt .Total
var SlippedAwayMsg = msg("slipped_away", "```"+`
  ──────── ·𖤍· ────────
   𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕨𝕒𝕟𝕕𝕖𝕣𝕤 𝕗𝕣𝕖𝕖
  ༺───────────────────༻
`+"```"+`
The goat has evaded another nabbing! {{.Count}} {{if eq .Count 1}}thief has{{else}}thieves have{{end}} clumsily attempted to snag it{{if .Loser}} from {{.Loser}}{{end}}. But the goat frolics freely, hoping to find its rightful home by {{.DeadlineAt}}.`)

// Sent when the holder left the server and nobody had entered, so the goat is
// now unheld and waiting for the next round's entrants.
// Fields: .Loser .Deadline .DeadlineAt
var EscapedUnheldMsg = msg("escaped_unheld", "```"+`
  ──────── ·𖤍· ────────
  𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕨𝕒𝕟𝕕𝕖𝕣𝕤 𝕗𝕣𝕖𝕖
  ༺───────────────────༻
`+"```"+`
The goat's latest keeper has fled the Kingdom, and so it frolics freely. Will it find its rightful home at last?
Surely no one will attempt to capture it before {{.DeadlineAt}}.`)

// Sent when a whole round passed with the goat unheld and nobody reaching for
// it, so it returns to the home an admin configured with /goat config home.
// Fields: .Holder .Free .Deadline .DeadlineAt
var HomeMsg = msg("home", "```"+`
  ──────── ·𖤍· ────────
    𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕘𝕠𝕖𝕤 𝕙𝕠𝕞𝕖
  ༺───────────────────༻
`+"```"+`
At last! After {{.Free}} of wandering freely, the goat has found its proper home. {{.Holder}} keeps it now. Surely no one would even attempt to swipe it off them! 'Twould be of great dishonour...`)

// Sent when an admin runs /goat reset: the streak is wiped and the goat is set
// loose again, belonging to nobody.
// Fields: .Actor .Minutes .Free .Deadline .DeadlineAt
var ResetMsg = msg("reset", "```"+`
  ──────── ·𖤍· ────────
       𝕒 𝕟𝕖𝕨 𝕠𝕣𝕕𝕖𝕣
  ༺───────────────────༻
`+"```"+`
{{.Actor}} has torn up the old order. Every streak is forgotten, and the goat once more roams free.
Surely no one will attempt to snatch it before {{.DeadlineAt}}, when its back is turned.`)

// ---------------------------------------------------------------------------
// PRIVATE REPLIES  (ephemeral: only the member who ran the command sees these)
// ---------------------------------------------------------------------------

// /goat steal succeeded. They are in the draw for this round.
// Fields: .Count .Deadline .DeadlineAt
var EnteredMsg = msg("entered", `You dared to steal the goat. But know this! Others may have attempted as well. And by {{.DeadlineAt}}, all shall be revealed.`)

// /goat steal when they already entered this round. One entry each, per the rules.
// Fields: .Count .Deadline .DeadlineAt
var AlreadyEnteredMsg = msg("already_entered", `You fool! There is only one goat to steal. And you may not try again, until the next goat keeper is declared at {{.DeadlineAt}}. Be patient.`)

// /goat steal by the current holder. They cannot steal from themselves.
// Fields: .Streak .Deadline .DeadlineAt
var YouHoldItMsg = msg("you_hold_it", `You fool! There is only one goat to steal, and you have already captured it. But until {{.DeadlineAt}}, sleep with one eye open. Others may be plotting to snatch it back.`)

// /goat steal before /goat setup has ever been run, so there is no round yet.
// Fields: none
var NoRoundMsg = msg("no_round", `The goat is nowhere to be found. Only the wisest folks in the Kingdom will know where to find it.`)

// An admin-only subcommand was run by somebody without Manage Server.
// Fields: none
var NotAdminMsg = msg("not_admin", `Forbidden attempt to command the goat's fate!`)

// /goat setup when a game is already running. Refuses, so a stray setup can
// never wipe an ongoing game. Points at /goat reset instead.
// Fields: .Holder .Minutes .Deadline .DeadlineAt
var AlreadyRunningMsg = msg("already_running", `The goat has already been set loose upon the Kingdom. No setup required. At {{.DeadlineAt}} we shall see who was the most cunning.
Run /goat reset if you wish to tear up the old order and begin again.`)

// /goat config confirmation.
// Fields: .Channel .Minutes
var ConfigDoneMsg = msg("config_done", `A successful command!`)

// ---------------------------------------------------------------------------
// READOUTS  (public replies to /goat status and /goat history)
// ---------------------------------------------------------------------------

// /goat status while somebody holds the goat.
// Fields: .Holder .Streak .Count .Deadline .DeadlineAt .HeldSince .Held .Total
var StatusMsg = msg("status", `The keeper of the goat is known as {{.Holder}}. They've held the creature for {{.Held}}.
But they must be on guard until {{.DeadlineAt}}, for others may reach for it...`)

// /goat status while the goat is unheld, because the holder left the server.
// Fields: .Count .Deadline .DeadlineAt .Free
var StatusUnheldMsg = msg("status_unheld", `The goat roams free! The beloved creature has been frolicking for {{.Free}}. Perhaps it will find its rightful home. Surely no one would be so nasty as to steal it for themselves.`)

// /goat history. .Lines is every HistoryLineMsg below, newest first, joined
// with newlines.
// Fields: .Lines .Total .Escapes .JourneysHome
var HistoryHeaderMsg = msg("history_header", "```"+`
  ──────── ·𖤍· ────────
      𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕝𝕖𝕕𝕘𝕖𝕣
  ༺───────────────────༻
`+"```"+`
In the goat's lifetime, much has occurred~
{{.Total}} {{if eq .Total 1}}thieving{{else}}thievings{{end}}, {{.Escapes}} successful {{if eq .Escapes 1}}escape{{else}}escapes{{end}} and {{.JourneysHome}} {{if eq .JourneysHome 1}}journey{{else}}journeys{{end}} home.

{{.Lines}}`)

// A date heading inside HistoryHeaderMsg's .Lines, printed once before the
// reigns that began on that day. Reigns are grouped by the bot machine's date.
// Fields: .From
var HistoryDayMsg = msg("history_day", `**{{.From}}**`)

// One row inside HistoryHeaderMsg's .Lines. Rendered once per past reign.
// Keep it to a single line.
// Fields: .User .From .To .Streak
var HistoryLineMsg = msg("history_line", `- {{.User}} held the goat from {{.From}} until {{.To}}.`)

// /goat history before anybody has lost the goat.
// Fields: .Total .Escapes .JourneysHome
var HistoryEmptyMsg = msg("history_empty", `The goat has never changed hands. The ledger begins with the first successful theft.`)

// /goat show by the current holder. Public.
// Fields: .Holder .Streak .HeldSince .Held
var ShowGoatMsg = msg("show_goat", "```"+`
⠀⣠⢄⢄⠀⢀⢔⠖⠲⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⣀⠡⠀⣱⠱⣎⢣⡤⢄⡡⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠈⠉⠁⢙⠋⠉⠟⠀⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣦⣦⢄
⠀⠀⠀⡞⡤⢬⣧⡤⠤⡀⠀⠀⠀⣀⠠⢄⣆⣤⡀⠀⠈⣿⡸
⠀⠀⢰⡀⠛⠾⠯⠃⠀⠈⠀⠀⠀⠀⠐⠛⠋⠀⢈⣲⠞⠋⠀
⠀⠀⠘⣧⠀⢀⠆⠀⠀⠀⠀⠀⠀⠀⠀⢀⠀⠀⠞⣸⠀⠀⠀
⠀⠀⠀⠸⣿⣮⠀⠀⢘⣦⠀⠀⠀⠀⣠⣾⠀⢊⣲⠏⠀⠀⠀
⠀⠀⠀⠀⠙⢻⣷⠀⣴⠛⠛⠿⠿⠿⠿⠛⡀⢠⡟⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠘⢻⠀⡏⠀⠀⠀⠀⠀⠀⠀⠡⡨⢷⡄⠀⠀⠀
⠀⠀⠀⠀⠀⠀⣇⣾⠃⠀⠀⠀⠀⠀⠀⠀⠀⣇⠻⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⣿⣼⡀⠀⠀⠀⠀⠀⠀⠀⢠⣟⡇⠀⠀⠀⠀
⠀⢠⠦⠶⠤⠞⡵⢩⡥⠴⠲⠪⠵⠶⠶⣶⢞⠿⠷⠶⠶⠶⠒
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠁⠀⠀⠀⠀⠀⠀
`+"```"+`
{{.Holder}} stole the goat {{.HeldSince}}.{{if .Streak}} No one else has dared reach for it in {{.Streak}} {{if eq .Streak 1}}round{{else}}rounds{{end}}.{{end}}`)

// /goat show by anybody else. Ephemeral.
// Fields: .Holder .Count .Deadline .DeadlineAt
var ShowNotHolderMsg = msg("show_not_holder", `The goat is held by another. {{.Holder}} has it. By typing /goat steal, you can attempt to take it for yourself. If the goat is to change hands, its next keeper will be declared {{.Deadline}}.`)

// ---------------------------------------------------------------------------
// WARNINGS  (things an admin needs to fix)
// ---------------------------------------------------------------------------

// Posted publicly instead of a theft announcement when the goat role could not
// be moved, which almost always means the bot's own role sits below the goat
// role and needs dragging above it. No transfer happened: the previous holder
// keeps the goat, so this must not congratulate anyone.
// Fields: .Winner .Holder .Reason
var RoleFailedMsg = msg("role_failed", "```"+`
  ──────── ·𖤍· ────────
  𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 𝕨𝕚𝕝𝕝 𝕟𝕠𝕥 𝕞𝕠𝕧𝕖
  ༺───────────────────༻
`+"```"+`
To much surprise, no thieving has taken place, despite {{.Count}} {{if eq .Count 1}}attempt{{else}}attempts{{end}}. {{if .Holder}}{{.Holder}} keeps the goat, merely by happenstance.{{else}}The goat remains unclaimed, merely by happenstance.{{end}}
Only the wisest in the Kingdom can decipher this strange and concerning message: {{.Reason}}.{{if not .Transient}}
Heed it, wise ones: the bot's own role must sit above the goat keeper role in Server Settings. Drag it higher, then run /goat resolve to settle this round properly.{{end}}`)

// Ephemeral. An admin command ran but no announcement channel is configured.
// Fields: .Channel .Minutes
var ChannelMissingMsg = msg("channel_missing", `Nowhere to announce it. Run /goat config channel:#somewhere first, then try again.`)

// Ephemeral. The goat role was deleted out from under the bot and could not be
// recreated.
// Fields: .Reason
var RoleMissingMsg = msg("role_missing", `The goat keeper role has vanished from the server! The goat bot could not recreate it, for reason: {{.Reason}}`)

// ---------------------------------------------------------------------------
// COMMAND PICKER TEXT
// ---------------------------------------------------------------------------
//
// What members read in Discord's slash-command autocomplete. Descriptions are
// capped at 100 characters. The command and option NAMES are not here: Discord
// requires those to be lowercase with no spaces, and changing one would change
// what players type, so they are fixed in commands.go.

var (
	DescSteal         = "Creep up on the goat and try to steal it this round"
	DescGoat          = "The goat"
	DescStatus        = "Where the goat is and how long is left"
	DescHistory       = "The ledger of past keepers, thievings and escapes"
	DescShow          = "Look upon the goat, and show it off if it is yours"
	DescSetup         = "Set the goat loose and begin the game (admin)"
	DescReset         = "Wipe the slate and set the goat loose again (admin)"
	DescResolve       = "End the current round right now (admin)"
	DescConfig        = "Set the announcement channel, round length and the goat's home (admin)"
	DescConfigHome    = "Who the goat returns to if nobody reaches for it"
	DescClearHome     = "Stop the goat returning to anybody (admin)"
	DescConfigChannel = "Where public announcements go"
	DescConfigMinutes = "Round length in minutes, applied from the next round"
)

// Shown when a command's reply slot above is still empty. Discord requires
// every command to get some reply, so this cannot be blank; it exists to make
// an unwritten slot obvious rather than to be read by players.
var EmptySlotNotice = "(no text written for this message yet)"
