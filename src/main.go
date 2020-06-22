package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Message is a base struct for messages
type Message struct {
	ID       string `json:"id"`
	ClientID string `json:"clientId"`
	Text     string `json:"text"`
	Operator string `json:"operator"`
}

func main() {
	amqpHost := os.Getenv("AMQP_HOST")
	if amqpHost == "" {
		amqpHost = "localhost"
	}
	isInit := make(chan string)
	go initPublisher("amqp://guest:guest@"+amqpHost+":5672", "test.billing", "fanout", isInit)
	<-isInit
	countMessages := 100000
	//prepare messages
	now := time.Now()
	for i := 0; i < countMessages; i++ {
		go func() {
			msg := Message{
				ID:       randomString(32, false),
				ClientID: randomString(32, false),
				Text:     randomString(170, false),
				Operator: randomString(32, false),
			}
			data, err := json.Marshal(msg)
			if err != nil {
				logrus.Error("[main] ", "Error json marshal: ", err)
			}
			Publish("test.billing", "", data)
		}()

	}
	after := time.Since(now).Seconds()
	logrus.Info("seconds: ", after)
	for {

	}

}

func sendRequest(reqURL string, method string, data []byte) ([]byte, error) {

	reqBody := bytes.NewBuffer(data)
	req, err := http.NewRequest(method, reqURL, reqBody)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error("[sendRequest] ", "Error send request: "+reqURL, err)
		return nil, err
	}
	defer resp.Body.Close()

	parsedResp, errParse := ioutil.ReadAll(resp.Body)
	if errParse != nil {
		logrus.Error("[sendRequest] ", "Error parse response body for req: "+reqURL, err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		logrus.Error("[sendRequest] ", "billing api error: ", string(parsedResp))
	}
	return parsedResp, nil
}

func randomString(n int, onlyDigits bool) string {
	var letter []rune
	if onlyDigits {
		letter = []rune("0123456789")
	} else {
		letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	}

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
