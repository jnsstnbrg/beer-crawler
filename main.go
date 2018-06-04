package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	b "github.com/larsha/bolaget.io-sdk-go"
)

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
	products, err := b.GetProducts(b.ProductQueryParams{
		Assortment:     "TSE",
		Limit:          100,
		ProductGroup:   "Öl",
		SalesStartFrom: today,
		SalesStartTo:   futureWeek,
	})
	if err != nil {
		log.Println(err)
		return "Error", err
	}

	// Check if products slice contains any items
	if len(products) > 0 {
		// URL to systembolaget, is later sent to Slack
		systembolagetURL := "https://www.systembolaget.se/sok-dryck/?assortmenttext=Sm%C3%A5%20partier&sellstartdatefrom=" + today + "&sellstartdateto=" + futureWeek + "&subcategory=%C3%96l&fullassortment=1"

		// Send to Slack
		err = sendToSlack(products, systembolagetURL)
		if err != nil {
			log.Println(err.Error())
			return "Error", err
		}
	}

	return "Done", nil
}

func sendToSlack(products []b.Product, systembolagetURL string) error {
	// Create buffer
	var buffer bytes.Buffer

	// Write topic(s) and url to buffer
	buffer.WriteString(":beers: *Nytt ölsläpp inom en vecka!* :beers: (" + products[0].SalesStart + ")\n")
	buffer.WriteString(systembolagetURL + "\n\n")
	buffer.WriteString("*Öl, Bryggeri, Pris, Storlek, ABV, Typ, Förpackning*\n")

	for _, product := range products {
		// Create product buffer
		var productBuffer bytes.Buffer

		// Write product specific information to buffer
		productBuffer.WriteString("*" + product.Name + " " + product.AdditionalName + "*, ")
		productBuffer.WriteString(product.Producer + ", ")
		productBuffer.WriteString(strconv.FormatFloat(product.Price.Amount, 'f', 2, 64) + " " + product.Price.Currency + ", ")
		productBuffer.WriteString(strconv.FormatFloat(product.VolumeInMilliliter, 'f', 0, 64) + " ml, ")
		productBuffer.WriteString(product.Alcohol + ", ")
		productBuffer.WriteString(product.Style + ", ")
		productBuffer.WriteString(product.Packaging + ", ")
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
