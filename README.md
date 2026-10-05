# Goat Bot
##### a discord bot, customly made, about goat theft

#### THE IDEA:
Admin sets the goat loose, belonging to nobody. All members of the server then have the opportunity to steal it for themselves, once per hour. One of the thieves is randomly selected to become the next 'goat holder' and so forth, though the goat may also slip every hand and wander free again. And a goat left wandering a whole round with nobody reaching for it finds its way to a home, if an admin has named one.

-----

#### FEATURES (v1):
One goat, one holder, one hour at a time. Everybody gets to type `/goat steal` once per round, and typing it again won't help you. The current holder can't steal from themselves, sadly. When the hour runs out, one of the thieves is picked at random and the goat changes hands. If nobody bothered to try, the holder keeps it and their streak goes up by one.

A goat belonging to nobody behaves differently. Reach for it and the usual draw happens, but leave it alone for a whole round and it goes home: `/goat config home:@somebody` names that person, and from then on an unclaimed goat ends up with them rather than wandering forever. They hold it like anybody else and can be stolen from. `/goat clear-home` takes the home away again, since a user picker can't be submitted empty. With no home set the goat simply keeps wandering, which is what it did before.

The goat itself is a real Discord role called '𓃵 𝕘𝕠𝕒𝕥 𝕜𝕖𝕖𝕡𝕖𝕣', and the bot is the only thing allowed to hand it out. Sticking a goat in your nickname does nothing, sorry.

`/goat status` tells you who has it and how long is left. `/goat history` shows the recent holders and how long each of them clung on. The whole game lives in one little `state.json` file, so the bot can be shut down, updated, and started back up without losing the goat. If it was asleep when an hour ran out, it settles that one round when it wakes up and starts fresh, rather than pretending to play the six hours it missed.

If the holder leaves the server while holding the goat, the goat 'escapes' and the next round's thieves get to recover it. Nobody gets credit for stealing from somebody who ran away.

-----

#### DEV SETUP:

Golang.
Docker.
Make.

Fill out `.env` w/ vars from `.env.example`.

#### ADD TO SERVER:

Then invite it to the server. OAuth2, URL Generator, tick `bot` and `applications.commands`, then tick Manage Roles, Send Messages and Embed Links. Open the link and pick your server.

In Server Settings, Roles, ensure the bot's own role is above the goat role.

Turn on Developer Mode (User Settings, Advanced), right click the server, Copy Server ID, and drop that into `.env` as `GUILD_ID`.

Then `make run`, and once it's up, `/goat config channel:#wherever` followed by `/goat setup`. The goat is now loose.

For leaving it up unattended, `make up` builds it and hands it to docker. `make logs` to watch it, `make down` to stop it. Run `make` on its own to see the lot.

`make up` again is also how you redeploy after changing anything. The save file lives in `./data` on your own disk, so the goat survives a rebuild.

-----

#### TESTING:
Unit testing, w/ discord mocked: `make test`. No token or internet needed.

`make lint` formats everything and runs `go vet` over it, which catches the daft mistakes the compiler lets through.

#### PREVIEWING THE BANNERS:

Fonts lie. A banner that lines up perfectly in your editor can come out crooked in Discord, because Discord's code blocks only render some characters as monospace and quietly swap the font for the rest.

Run `/goat preview` in whatever channel you like and the bot will post every banner message there, filled with dummy names and times, each one labelled. Then look at it on desktop and on your phone and see the truth. Nobody gets pinged, the fake mentions are inert. Add `all:True` to include the messages that have no banner.

It also tells you, privately, about any characters sitting outside the ranges Discord keeps monospace. Box drawing and braille are safe. Fancy lettering like 𝕥𝕙𝕖 𝕘𝕠𝕒𝕥 is not, nor are ornaments like 𖤍 or ༺ ༻, so those will drift no matter how neat they look here.

-----

#### COMMANDS:
```txt
/goat steal                         everyone    try to take the goat this hour
/goat status                        everyone    who has it, how long is left
/goat history                       everyone    recent holders
/goat show                          everyone    the goat, if it's yours
/goat setup                         admin       set the goat loose
/goat reset                         admin       wipe the slate, set it loose again
/goat resolve                       admin       end this round right now
/goat config [channel] [minutes] [home]
                                    admin       channel, round length and the goat's home
/goat clear-home                    admin       stop the goat returning to anybody
/goat preview [all]                 admin       post every banner here to check how it looks
```




