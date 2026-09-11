# Return the Goat
##### A discord bot, customly made for Netherwilds discord server.
```txt
===========================================================================================
      J O I N - T H E - P E A S A N T - U P R I S I N G - O C T - N I N T H - 2 0 2 6
===========================================================================================
```

#### THE IDEA:
Admin will select the first goat holder (role). But fear not! All members of the server will have the opportunity to steal the goat for themselves, once per hour. One of the theives will be randomly selected to become the next 'goat holder' and so forth.

-----

#### FEATURES (v1):
One goat, one holder, one hour at a time. Everybody gets to type `/steal` once per round, and typing it again won't help you. The current holder can't steal from themselves, sadly. When the hour runs out, one of the theives is picked at random and the goat changes hands. If nobody bothered to try, the holder keeps it and their streak goes up by one.

The goat itself is a real Discord role called '🐐 Goat Holder', and the bot is the only thing allowed to hand it out. Sticking a goat in your nickname does nothing, sorry.

`/goat status` tells you who has it and how long is left. `/goat history` shows the recent holders and how long each of them clung on. The whole game lives in one little `state.json` file, so the bot can be shut down, updated, and started back up without losing the goat. If it was asleep when an hour ran out, it settles that one round when it wakes up and starts fresh, rather than pretending to play the six hours it missed.

If the holder leaves the server while holding the goat, the goat 'escapes' and the next round's theives get to recover it. Nobody gets credit for stealing from somebody who ran away.

-----

#### DEV SETUP:
You'll need Go. Everything else the bot needs, it fetches itself with `go mod tidy`.

Make an application over at discord.com/developers/applications, add a bot to it, and hit Reset Token to get a token out of it. Copy `.env.example` to `.env` and paste the token in there. Leave both of the 'privileged intents' switches off, this bot doesn't want either of them.

Then invite it to the server. OAuth2, URL Generator, tick `bot` and `applications.commands`, then tick Manage Roles, Send Messages and Embed Links. Open the link it builds you and pick the server.

Now the important bit, and the one that goes wrong: in Server Settings, Roles, drag the bot's own role ABOVE the goat role. Discord won't let a bot touch any role sitting higher than itself, so if you skip this the goat simply refuses to move and the bot will start complaining in the channel about it.

Last thing, the bot needs to know which server it's in. Turn on Developer Mode (User Settings, Advanced), right click the server, Copy Server ID, and drop that into `.env` as `GUILD_ID`.

Then `go run .`, and once it's up, `/goat config channel:#wherever` followed by `/goat setup`. The goat is now loose.

There's no web server and no ports to open, by the way. The bot phones out to Discord and keeps the line open, so it runs happily from a laptop behind a router. It only plays the game while it's actually running though. For leaving it up unattended there's a `compose.yaml`, so `docker compose up -d --build` will keep it alive through crashes and reboots, and keeps the save file in `./data`.

-----

#### TESTING:
`go test ./...` and that's it. No token, no internet, no Discord. There's a fake stand-in for Discord in `fake_discord_test.go`, so the tests can make somebody leave the server, or make a role assignment fail on purpose, and check the bot does the sensible thing.

The lottery gets its own test that runs ten thousand rounds and makes sure all four theives win about a quarter of the time each, because a rigged goat is no fun.

The other one worth knowing about: if the bot can't actually move the role, it is not allowed to announce that anybody stole anything. It keeps the goat where it was and moans about permissions instead. There's a test that holds it to that.

For trying it out in a real server without waiting around all day, `/goat config minutes:2` makes the rounds two minutes long, and `/goat resolve` ends the current one on the spot.

-----

#### THE MESSAGES:
Everything the bot ever says out loud lives in `strings.go`, and every single one starts out blank. Write whatever you like in them. Anything left blank simply doesn't get sent, so an unfinished file just makes for a quiet goat rather than one spouting placeholder nonsense.

Times get handed to you as Discord's own timestamp things, so `{{.Deadline}}` comes out as a countdown that ticks along by itself and shows in each person's own timezone. ASCII art needs to go inside a code fence (three backticks) or Discord's font will mangle it, and a message can't go over 2000 characters.

-----

#### COMMANDS:
```txt
/steal                              everyone    get in this hour's draw
/goat status                        everyone    who has it, how long is left
/goat history                       everyone    recent holders
/goat setup [member]                admin       start the game
/goat reset [member]                admin       start over on a new holder
/goat resolve                       admin       end this round right now
/goat config [channel] [minutes]    admin       channel and round length
```
Admin means Manage Server or Administrator. A new round length only kicks in from the next round, so changing it can't yank the deadline out from under a round already in progress.




