# Goat Bot
##### a discord bot, customly made, about goat theft

#### THE IDEA:
Admin will select the first goat holder (role). But fear not! All members of the server will have the opportunity to steal the goat for themselves, once per hour. One of the thieves will be randomly selected to become the next 'goat holder' and so forth.

-----

#### FEATURES (v1):
One goat, one holder, one hour at a time. Everybody gets to type `/steal` once per round, and typing it again won't help you. The current holder can't steal from themselves, sadly. When the hour runs out, one of the thieves is picked at random and the goat changes hands. If nobody bothered to try, the holder keeps it and their streak goes up by one.

The goat itself is a real Discord role called '𓃵 𝕘𝕠𝕒𝕥 𝕥𝕙𝕚𝕖𝕗', and the bot is the only thing allowed to hand it out. Sticking a goat in your nickname does nothing, sorry.

`/goat status` tells you who has it and how long is left. `/goat history` shows the recent holders and how long each of them clung on. The whole game lives in one little `state.json` file, so the bot can be shut down, updated, and started back up without losing the goat. If it was asleep when an hour ran out, it settles that one round when it wakes up and starts fresh, rather than pretending to play the six hours it missed.

If the holder leaves the server while holding the goat, the goat 'escapes' and the next round's thieves get to recover it. Nobody gets credit for stealing from somebody who ran away.

-----

#### DEV SETUP:

Golang.
Docker.

Fill out `.env` w/ vars from `.env.example`.

#### ADD TO SERVER:

Then invite it to the server. OAuth2, URL Generator, tick `bot` and `applications.commands`, then tick Manage Roles, Send Messages and Embed Links. Open the link and pick your server.

In Server Settings, Roles, ensure the bot's own role is above the goat role.

Turn on Developer Mode (User Settings, Advanced), right click the server, Copy Server ID, and drop that into `.env` as `GUILD_ID`.

Then `go run ./cmd/goat-bot`, and once it's up, `/goat config channel:#wherever` followed by `/goat setup`. The goat is now loose.

For leaving it up unattended there's a `compose.yaml`, so run `docker compose up -d --build`.

-----

#### TESTING:
Unit testing, w/ discord mocked: `go test ./...`

If you testing manually in a discord server, `/goat config minutes:2` makes the rounds two minutes long, and `/goat resolve` ends the current one. These are 'admin' only commands, meaning you need Manage Server or Administrator discord server permission. 

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




