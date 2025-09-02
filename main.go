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
)

type Match struct {
	HomeTeam      string
	AwayTeam      string
	DateAndTime   string
	Round         string
	Competition   string
	DateTimeStamp string
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

	pageData := getSourcePage("https://hcdynamo.cz/matches/MUZ", client)
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
	q.Add("next_match_date_place", match.DateAndTime)
	q.Add("next_match_round", match.Round)
	q.Add("next_match_competition", match.Competition)
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
	defer matchesPage.Body.Close()
	if matchesPage.StatusCode != 200 {
		log.Error(fmt.Sprintf("Requesting the pages failed, got errorcode: %d", matchesPage.StatusCode))
	}
	return body
}

func parseNextMatch(pageData []byte) Match {
	doc, parseErr := goquery.NewDocumentFromReader(bytes.NewReader(pageData))
	logErrorIfPresentAndExit(parseErr)
	log.Debug(doc.Html())
	dateAndPlace := ""
	homeTeam := ""
	awayTeam := ""
	round := ""
	competition := ""
	dateTimeStamp := ""
	matchSection := doc.Find(`section[aria-label="Nejbližší zápas"]`)
	if matchSection.Length() == 0 {
		logErrorIfPresentAndExit(fmt.Errorf("sekce s nejbližším zápasem nebyla nalezena"))
	}

	matchItem := matchSection.Find("li").First()
	if matchItem.Length() == 0 {
		logErrorIfPresentAndExit(fmt.Errorf("žádný zápas nebyl nalezen"))
	}

	firstDiv := matchItem.Find("div").First()
	spans := firstDiv.Find("span")
	if spans.Length() >= 2 {
		round = strings.TrimSpace(spans.Eq(0).Text())
		competition = strings.TrimSpace(spans.Eq(1).Text())
	}

	timeElement := matchItem.Find("time")
	if timeElement.Length() > 0 {
		if datetime, exists := timeElement.Attr("datetime"); exists {
			dateTimeStamp = datetime
		}
		dateAndPlace = strings.TrimSpace(timeElement.Text())
	}

	// Extrahuje týmy z alt atributů obrázků
	var teams []string
	matchItem.Find("img[alt]").Each(func(i int, img *goquery.Selection) {
		if alt, exists := img.Attr("alt"); exists {
			// Vyčistí alt text od "Logo " prefixu
			teamName := strings.TrimPrefix(alt, "Logo ")
			teamName = strings.TrimSpace(teamName)
			if teamName != "" {
				teams = append(teams, teamName)
			}
		}
	})

	if len(teams) >= 2 {
		homeTeam = teams[0]
		awayTeam = teams[1]
	}
	return Match{HomeTeam: homeTeam, AwayTeam: awayTeam, DateAndTime: dateAndPlace, Round: round, Competition: competition, DateTimeStamp: dateTimeStamp}
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
	return fmt.Sprintf("Round: %s, Competition: %s, Home team: %s, Away team: %s, DateTimeStamp %s, TimeStamp: %s", m.Round, m.Competition, m.HomeTeam, m.AwayTeam, m.DateAndTime, m.DateTimeStamp)
}
