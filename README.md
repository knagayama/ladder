# Ladder
Ladder resolver -- intended for resolving challenges for Spladder-like Splatoon tournaments.

## Usage

You need access to the challenge form results spreadsheet.

0. Copy-paste the current prefs and teams to the prefs.json and teams.json sheets respectively.

1. Save credentials.json to the same dir. Easiest way is to get it from https://developers.google.com/sheets/api/quickstart/go

2. Set the currentRound const.

3. $ go run ladder.go spreadheets.go

You're done!
