package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"golang.org/x/text/encoding/charmap"
)

type Match struct {
	HomeTeam     string
	AwayTeam     string
	DateAndPlace string
	Time         string
}

func main() {
	godotenv.Load()
	zivyObrazKey := os.Getenv("ZIVYOBRAZ_KEY")
	skipZivyObraz := os.Getenv("SKIP_ZIVYOBRAZ")
	if skipZivyObraz == "" && zivyObrazKey == "" {
		log.Fatal("The ZIVYOBRAZ_KEY variable is not set.")
	}
	client, err := createClient()
	logErrorIfPresentAndExit(err)

	pageData := getSourcePage("https://hcdynamo.cz/zapasy.asp", client)
	log.Debugln("Got the page for parsing, converting to parsable document")
	match := parseNextMatch(pageData)
	log.Info(match)

	if skipZivyObraz == "" {
		publishToZivyObraz(match, zivyObrazKey, client)
	} else {
		log.Info("Variable SKIP_ZIVYOBRAZ is set, skipping publishing to zivyobraz.cz")
	}
	log.Info("Done!")
}

func publishToZivyObraz(match Match, zivyObrazKey string, client *http.Client) {
	log.Info("Publishing to zivyobraz")
	req, pubReqErr := http.NewRequest("GET", "https://in.zivyobraz.eu", nil)
	logErrorIfPresentAndExit(pubReqErr)
	q := req.URL.Query()
	q.Add("import_key", zivyObrazKey)
	q.Add("next_match_home_team", match.HomeTeam)
	q.Add("next_match_away_team", match.AwayTeam)
	q.Add("next_match_teams_full_string", fmt.Sprintf("%s - %s", match.HomeTeam, match.AwayTeam))
	q.Add("next_match_date_place", match.DateAndPlace)
	q.Add("next_match_time", match.Time)
	req.URL.RawQuery = q.Encode()
	res, publishResErr := client.Do(req)
	logErrorIfPresentAndExit(publishResErr)
	if res.StatusCode != 200 {
		log.Fatalf("Publishing to zivyobraz failed, got errorcode: %d", res.StatusCode)
	}
}

func getSourcePage(url string, client *http.Client) []byte {
	req, _ := http.NewRequest("GET", url, strings.NewReader(""))
	matchesPage, loadErr := client.Do(req)
	logErrorIfPresentAndExit(loadErr)
	body, readErr := io.ReadAll(matchesPage.Body)
	logErrorIfPresentAndExit(readErr)
	dec := charmap.Windows1250.NewDecoder()
	output, encodingConversionErr := dec.Bytes(body)
	logErrorIfPresentAndExit(encodingConversionErr)
	defer matchesPage.Body.Close()
	if matchesPage.StatusCode != 200 {
		log.Error(fmt.Sprintf("Requesting the pages failed, got errorcode: %d", matchesPage.StatusCode))
	}
	return output
}

func parseNextMatch(pageData []byte) Match {
	doc, parseErr := goquery.NewDocumentFromReader(bytes.NewReader(pageData))
	logErrorIfPresentAndExit(parseErr)
	game := doc.Find(".subpage .game").First()
	log.Debug(game.Html())
	dateAndPlace := ""
	homeTeam := ""
	awayTeam := ""
	game.ChildrenFiltered(".game_detail").Children().Each(func(i int, selection *goquery.Selection) {
		if i == 0 {
			content, _ := selection.Html()
			dateAndPlace = strings.TrimSpace(content)
			log.Debugf("Date and place string: %s", dateAndPlace)
		}
		if i == 1 {
			selection.Children().Each(func(i int, teams *goquery.Selection) {
				log.Debugf("Team [%d]: %s", i, teams.Text())
				if i == 0 {
					homeTeam = teams.Text()
				}
				if i == 1 {
					awayTeam = teams.Text()
				}
			})
		}
	})
	time := game.Find(".score_time").First().Text()
	return Match{HomeTeam: homeTeam, AwayTeam: awayTeam, DateAndPlace: dateAndPlace, Time: time}
}

func createClient() (*http.Client, error) {
	log.Debugln("Creating http client for connections")
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Got error while creating cookie jar %s", err.Error())
		return nil, err
	}
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Jar: jar,
	}
	return httpClient, nil
}

func logErrorIfPresentAndExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (m Match) String() string {
	return fmt.Sprintf("Home team: %s, Away team: %s, DateAndPlace: %s, Time: %s", m.HomeTeam, m.AwayTeam, m.DateAndPlace, m.Time)
}
