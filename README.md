# Dynamo next match scraper

1) scrapes hcdynamo.cz page for next match
2) extracts the data of next match
3) sends data to zivyobraz service

## Configuration

Rename .env.example to .env and set the proper zivyobraz import key in  `ZIVYOBRAZ_KEY` variable

If you set `SKIP_ZIVYOBRAZ` variable, the publishing to ZO will be skipped.

Also you can customize log level(`DEBUG|INFO|WARNING|ERROR|FATAL`)

## Sent ZivyObraz values

* `next_match_home_team` - home team
* `next_match_away_team` - away team
* `next_match_teams_full_string` - all data formatted in one string
* `next_match_date_place` - match date (formatted)
* `next_match_time` - match time (formatted)

#zadynamo
