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
	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/charmap"
)

func main() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	godotenv.Load()
	zivyObrazKey := os.Getenv("ZIVYOBRAZ_KEY")
	skipZivyObraz := os.Getenv("SKIP_ZIVYOBRAZ")
	if skipZivyObraz == "" && zivyObrazKey == "" {
		log.Fatal("The ZIVYOBRAZ_KEY variable is not set.")
	}
	client, err := createClient()
	logErrorIfPresentAndExit(err)
	req, _ := http.NewRequest("GET", "https://hcdynamo.cz/zapasy.asp", strings.NewReader(""))
	//matchesPage, loadErr := client.Get("https://hcdynamo.cz/zapasy.asp")
	// req.Header.Add("Accept-Charset", "utf-8")
	matchesPage, loadErr := client.Do(req)
	//os.Create("page.html")
	body, _ := io.ReadAll(matchesPage.Body)
	// os.WriteFile("page.html", body, 0644)
	logErrorIfPresentAndExit(loadErr)
	//converting to Windows1250, because thats the encoding that ASP.NET server uses
	dec := charmap.Windows1250.NewDecoder()
	output, _ := dec.Bytes(body)
	defer matchesPage.Body.Close()
	if matchesPage.StatusCode != 200 {
		log.Error(fmt.Sprintf("Requesting the pages failed, got errorcode: %d", matchesPage.StatusCode))
	}
	log.Debugln("Got the page for parsing, converting to parsable document")
	doc, parseErr := goquery.NewDocumentFromReader(bytes.NewReader(output))
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

	log.Infof("Home team: %s, Away team: %s, DateAndPlace: %s, Time: %s", homeTeam, awayTeam, dateAndPlace, time)
	//game.ChildrenFiltered(".game_detail").First()

	if skipZivyObraz == "" {
		log.Info("Publishing to zivyobraz")
		req, pubReqErr := http.NewRequest("GET", "https://in.zivyobraz.eu", nil)
		logErrorIfPresentAndExit(pubReqErr)
		q := req.URL.Query()
		q.Add("import_key", zivyObrazKey)
		q.Add("next_match_home_team", homeTeam)
		q.Add("next_match_away_team", awayTeam)
		q.Add("next_match_teams_full_string", fmt.Sprintf("%s - %s", homeTeam, awayTeam))
		q.Add("next_match_date_place", dateAndPlace)
		q.Add("next_match_time", time)
		req.URL.RawQuery = q.Encode()
		res, publishResErr := client.Do(req)
		logErrorIfPresentAndExit(publishResErr)
		if res.StatusCode != 200 {
			log.Fatalf("Publishing to zivyobraz failed, got errorcode: %d", res.StatusCode)
		}
	} else {
		log.Info("Variable SKIP_ZIVYOBRAZ is set, skipping publishing to zivyobraz.cz")
	}
	log.Info("Done!")
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
