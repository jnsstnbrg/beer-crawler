package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type price struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type customTime struct {
	time.Time
}

type product struct {
	Number             string     `json:"nr"`
	ArticleID          int32      `json:"article_id"`
	ArticleNumber      int32      `json:"article_nr"`
	Name               string     `json:"name"`
	AdditionalName     string     `json:"additional_name"`
	Price              price      `json:"price"`
	VolumeInMilliliter int        `json:"volume_in_milliliter"`
	PricePerLiter      float64    `json:"price_per_liter"`
	SalesStart         customTime `json:"sales_start"`
	Type               string     `json:"type"`
	Style              string     `json:"style"`
	Packing            string     `json:"packaging"`
	Producer           string     `json:"producer"`
	Alcohol            string     `json:"alcohol"`
}

type slackRequest struct {
	Text string `json:"text"`
}

func main() {
	lambda.Start(Handler)
}

// Handler is a lambda handler function
func Handler() (string, error) {
	// Save today's and a week ahead's date in variables
	today := time.Now().Format("2006-01-02")
	futureWeek := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	// URL to where the json data exists
	url := "https://bolaget.io/v1/products?sales_start_from=" + today + "&sales_start_to=" + futureWeek + "&limit=100&product_group=Öl&assortment=TSE"

	// URL to systembolaget, is later sent to Slack
	systembolagetURL := "https://www.systembolaget.se/sok-dryck/?assortmenttext=Sm%C3%A5%20partier&sellstartdatefrom=" + today + "&sellstartdateto=" + futureWeek + "&subcategory=%C3%96l&fullassortment=1"

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	// Unmarshal into a products slice
	var products []product
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	json.Unmarshal(body, &products)

	// Check if products slice contains any items
	if len(products) > 0 {
		// Send to Slack
		err = sendToSlack(products, systembolagetURL)
		if err != nil {
			log.Println(err.Error())
		}
	}

	return "Done", nil
}

func sendToSlack(products []product, systembolagetURL string) error {
	// Create buffer
	var buffer bytes.Buffer

	// Write topic(s) and url to buffer
	buffer.WriteString(":beers: *Nytt ölsläpp inom en vecka!* :beers: (" + products[0].SalesStart.Format("2006-01-02") + ")\n")
	buffer.WriteString(systembolagetURL + "\n\n")
	buffer.WriteString("*Öl, Bryggeri, Pris, Storlek, ABV, Typ, Förpackning*\n")

	for _, product := range products {
		// Create product buffer
		var productBuffer bytes.Buffer

		// Write product specific information to buffer
		productBuffer.WriteString("*" + product.Name + " " + product.AdditionalName + "*, ")
		productBuffer.WriteString(product.Producer + ", ")
		productBuffer.WriteString(strconv.FormatFloat(product.Price.Amount, 'f', 1, 64) + " " + product.Price.Currency + ", ")
		productBuffer.WriteString(strconv.Itoa(product.VolumeInMilliliter) + " ml, ")
		productBuffer.WriteString(product.Alcohol + ", ")
		productBuffer.WriteString(product.Style + ", ")
		productBuffer.WriteString(product.Packing + ", ")
		productBuffer.WriteString("\n")

		// Write product buffer to main buffer
		buffer.WriteString(productBuffer.String())
	}

	// Marshal buffer to json
	data, err := json.Marshal(slackRequest{buffer.String()})
	if err != nil {
		return err
	}

	// Create request
	req, err := http.NewRequest("POST", os.Getenv("SLACK_URL"), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// Send request to Slack
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (ct *customTime) UnmarshalJSON(b []byte) (err error) {
	// Trim away "
	s := strings.Trim(string(b), "\"")

	// Parse time
	ct.Time, err = time.Parse("2006-01-02", s)
	return
}
