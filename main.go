package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type slackRequest struct {
	Text string `json:"text"`
}

type beverage struct {
	ProductID         string  `json:"ProductId"`
	SubCategory       string  `json:"SubCategory"`
	ProductNameBold   string  `json:"ProductNameBold"`
	ProductNameThin   string  `json:"ProductNameThin"`
	ProducerName      string  `json:"ProducerName"`
	Price             float64 `json:"Price"`
	Volume            float64 `json:"Volume"`
	AlcoholPercentage float64 `json:"AlcoholPercentage"`
	Style             string  `json:"Style"`
	BottleTextShort   string  `json:"BottleTextShort"`
	SellStartDate     string  `json:"SellStartDate"`
}

type doc struct {
	Beverages []beverage `json:"Hits"`
}

func main() {
	// Save today's and a week ahead's date in variables
	today := time.Now().Format("2006-01-02")
	futureWeek := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	URL, err := url.Parse("https://api-extern.systembolaget.se/product/v1/product/search")
	if err != nil {
		panic("boom")
	}

	parameters := url.Values{}
	parameters.Add("AssortmentText", "Lokalt & Småskaligt")
	parameters.Add("SellStartDateFrom", today)
	parameters.Add("SellStartDateTo", futureWeek)
	parameters.Add("SubCategory", "Öl")
	URL.RawQuery = parameters.Encode()

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", URL.String(), nil)

	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Add("Ocp-Apim-Subscription-Key", os.Getenv("SUBSCRIPTION_KEY"))

	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	doc := doc{}

	jsonErr := json.Unmarshal(body, &doc)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
	}

	// Check if products slice contains any items
	if len(doc.Beverages) > 0 {
		// URL to systembolaget, is later sent to Slack
		systembolagetURL := "https://www.systembolaget.se/sok-dryck/?assortmenttext=Sm%C3%A5%20partier&sellstartdatefrom=" + today + "&sellstartdateto=" + futureWeek + "&subcategory=%C3%96l&fullassortment=1"

		// Send to Slack
		err = sendToSlack(doc.Beverages, systembolagetURL)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
}

func sendToSlack(beverages []beverage, systembolagetURL string) error {
	// Create buffer
	var buffer bytes.Buffer

	// Write topic(s) and url to buffer
	buffer.WriteString(":beers: *Nytt ölsläpp inom en vecka!* :beers: (" + beverages[0].SellStartDate + ")\n")
	buffer.WriteString(systembolagetURL + "\n\n")
	buffer.WriteString("*Öl, Bryggeri, Pris, Storlek, ABV, Typ, Förpackning*\n")

	for _, product := range beverages {
		// Create product buffer
		var productBuffer bytes.Buffer

		// Write product specific information to buffer
		productBuffer.WriteString("*" + product.ProductNameBold + "*")
		if product.ProductNameThin != "" {
			productBuffer.WriteString(" *" + product.ProductNameThin + "*")
		}
		productBuffer.WriteString(", ")
		productBuffer.WriteString(product.ProducerName + ", ")
		productBuffer.WriteString(strconv.FormatFloat(product.Price, 'f', 2, 64) + " SEK, ")
		productBuffer.WriteString(strconv.FormatFloat(product.Volume, 'f', 0, 64) + " ml, ")
		productBuffer.WriteString(strconv.FormatFloat(product.AlcoholPercentage, 'f', 0, 64) + "%, ")
		productBuffer.WriteString(product.Style + ", ")
		productBuffer.WriteString(product.BottleTextShort + ", ")
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
