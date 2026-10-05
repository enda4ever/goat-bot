# Privacy Policy for Goat Bot

Last updated: 1 October 2026

-----

#### What identifying data is stored:
For functional reasons, your Discord user ID is stored if you've ever held the goat or tried to steal it. Also, the hour you got the goat, the hour you lost it, and how many hourly rounds running you managed to keep the goat. That is all.

An admin can also name one person as the goat's home, which is where the goat ends up if a whole round passes with nobody reaching for it. That person's user ID is stored from the moment they are named, before they have held the goat or tried to steal it, and it stays there until an admin names somebody else or runs `/goat clear-home`.

A user ID is a public identifying number that Discord gives you when you sign up. It never changes, and anybody holding one can look up whose profile it belongs to. So this isn't anonymous data. Any bot which needs to give out "roles" to users needs access to their corresponding user IDs.

-----

#### How long this data is stored:
The reign records keep the last fifty times the goat changed hands, and no more.

The list of who tried to steal this hour is wiped at the end of every hour, win or lose.

-----

#### Where the data is stored:
One `state.json` file on whichever machine is running the bot. It doesn't go anywhere else. The only thing on the other end of the wire is Discord itself, being told to move a role and post a message.

-----

#### You can have your user ID removed from storage:
Ask whoever runs the bot and they'll take your rows out of the history file by hand. There's no command for it yet. Leaving the server doesn't clear anything on its own, so if you want it gone you do have to say so.

-----

#### More questions are welcome:
team.4ever.info@gmail.com
