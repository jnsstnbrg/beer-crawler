package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
)

type release struct {
	Time  time.Time `json:"time"`
	Beers []beer    `json:"beers"`
	URL   string    `json:"url"`
}

type beer struct {
	Title    string `json:"title"`
	Brewery  string `json:"brewery"`
	Price    string `json:"price"`
	Size     string `json:"size"`
	ABV      string `json:"abv"`
	BeerType string `json:"beerType"`
	Country  string `json:"country"`
}

type slackRequest struct {
	Text string `json:"text"`
}

func main() {
	lambda.Start(Handler)
}

// Handler is a lambda handler function
func Handler() (string, error) {
	resp, err := getDocument("https://systemizr.se/release/")
	if err != nil {
		return "Error", err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return "Error", err
	}
	defer resp.Body.Close()

	var replacer = strings.NewReplacer(" ", "")

	var releases []release

	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		topicDates := strings.Split(replacer.Replace(s.Text()), "\n")
		topicYear := strings.TrimSpace(topicDates[1])

		s.NextUntil("h2, h3").Each(func(i int, r *goquery.Selection) {
			line := strings.TrimSpace(r.Text())
			if len(line) > 0 {
				releaseDates := strings.Split(line, " ")
				releaseMonth := releaseDates[1]
				releaseDay := releaseDates[0]

				if len(releaseDay) == 1 {
					releaseDay = "0" + releaseDay
				}

				t, err := time.Parse("2006-January-02", topicYear+"-"+convertMonth(strings.ToLower(releaseMonth))+"-"+releaseDay)
				if err != nil {
					log.Println(err.Error())
				}

				if inTimeSpan(time.Now(), time.Now().AddDate(0, 0, 7), t) {
					r.Find("a").Each(func(i int, a *goquery.Selection) {
						href, ok := a.Attr("href")
						if ok {
							beers, err := getBeers(href)
							if err != nil {
								log.Println(err.Error())
							}

							if len(beers) > 0 {
								releases = append(releases, release{t, beers, href})
							}
						}
					})
				}
			}
		})
	})

	sort.Slice(releases, func(i, j int) bool { return releases[i].Time.Before(releases[j].Time) })

	for _, release := range releases {
		err = sendToSlack(release)
		if err != nil {
			log.Println(err.Error())
		}
	}

	return "Done", nil
}

func sendToSlack(release release) error {
	// Create buffer
	var buffer bytes.Buffer

	buffer.WriteString(":beers: *Nytt ölsläpp inom en vecka!* :beers: (" + release.Time.Format("2006-01-02") + ")\n")
	buffer.WriteString(release.URL + "\n\n")
	buffer.WriteString("*Öl, Bryggeri, Pris, Storlek, ABV, Typ, Land*\n")
	for _, beer := range release.Beers {
		line := "*" + beer.Title + "*, " + beer.Brewery + ", " + beer.Price + "kr, " + beer.Size + ", " + beer.ABV + ", " + beer.BeerType + ", " + beer.Country + "\n"
		buffer.WriteString(line)
	}

	data, err := json.Marshal(slackRequest{buffer.String()})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", os.Getenv("SLACK_URL"), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func getBeers(url string) ([]beer, error) {
	resp, err := getDocument(url)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var beers []beer

	doc.Find("div.beer-table").Find("div.beer-row-new").Each(func(i int, s *goquery.Selection) {
		title := strings.Split(strings.TrimSpace(s.Find("span.title").Text()), "\n")[0]
		brewery := strings.Split(strings.TrimSpace(s.Find("span.brewery-title").Text()), "\n")[0]
		price := strings.Split(strings.TrimSpace(s.Find("span.price").Text()), "\n")[0]
		size := strings.Split(strings.TrimSpace(s.Find("div.pack-info span.size").Text()), "\n")[0]
		abv := strings.Split(strings.TrimSpace(s.Find("div.right-left-col span.abv").Text()), "\n")[0]
		beerType := strings.Split(strings.TrimSpace(s.Find("div.right-left-col span.value").Text()), "\n")[0]
		country := strings.Split(strings.TrimSpace(s.Find("div.right-left-col div.value").Text()), "\n")[0]

		beers = append(beers, beer{title, brewery, price, size, abv, beerType, country})
	})

	return beers, nil
}

func getDocument(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

func convertMonth(month string) string {
	switch month {
	case "januari":
		return "January"
	case "februari":
		return "February"
	case "mars":
		return "March"
	case "april":
		return "April"
	case "maj":
		return "May"
	case "juni":
		return "June"
	case "juli":
		return "July"
	case "augusti":
		return "August"
	case "september":
		return "September"
	case "oktober":
		return "October"
	case "november":
		return "November"
	case "december":
		return "December"
	default:
		return "Not a valid month"
	}
}
