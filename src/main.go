package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Message is a base struct for messages
type Message struct {
	ID       string `db:"id" json:"id"`
	ClientID string `db:"clientId" json:"clientId"`
	Text     string `db:"text" json:"text"`
	Operator string `db:"operator" json:"operator"`
}

var db *sqlx.DB

func main() {
	initAMQP()
	initDB()

	gin.SetMode("release")
	r := gin.New()
	r.GET("/handle/:count", func(c *gin.Context) {
		count := c.Param("count")
		generateMessages(count)
	})
	host := ":14500"
	logrus.Info("running http server on host: ", host)
	r.Run(host)
}

func initAMQP() {
	amqpHost := os.Getenv("AMQP_HOST")
	if amqpHost == "" {
		amqpHost = "localhost"
	}
	isInit := make(chan string)
	go initPublisher("amqp://guest:guest@"+amqpHost+":5672", "test.billing", "fanout", isInit)
	<-isInit
}

func initDB() {
	var err error
	connected := false
	defaultURI := "user=postgres password=VjMaexz$rF dbname=billing sslmode=disable"
	envURI := os.Getenv("SQL_URI")
	if envURI == "" {
		envURI = defaultURI
	}
	for !connected {
		db, err = sqlx.Connect("postgres", envURI)
		if err != nil {
			logrus.Error("failed connect to database:", envURI, " ", err, " try reconnect after 20 seconds")
			time.Sleep(20 * time.Second)
			continue
		} else {
			connected = true
		}
	}
	logrus.Info("success connect to database: ", envURI)
}

func generateMessages(count string) {
	data := []Message{}
	minLen := 50
	maxLen := 1000

	err := db.Select(&data, `SELECT id as "clientId", tariff as "operator" FROM client LIMIT `+count)
	if err != nil {
		logrus.Error("err get client data: ", err)
		return
	}
	//prepare messages
	logrus.Info("prepare messages: ", count)
	now := time.Now()
	for i := 0; i < len(data); i++ {
		msg := data[i]
		go func() {
			// prepare message with random length
			msg.ID = NewId()
			rand.Seed(time.Now().UnixNano())
			msgLen := rand.Intn(maxLen-minLen+1) + minLen
			msg.Text = randomString(msgLen, false)

			// send message to amqp exchange
			data, err := json.Marshal(msg)
			if err != nil {
				logrus.Error("[main] ", "Error json marshal: ", err)
			}
			Publish("test.billing", "", data)
		}()

	}
	after := time.Since(now).Seconds()
	logrus.Info("seconds: ", after)
}

// NewId returns new UUID
func NewId() (res string) {
	uuidVal := uuid.New().String()
	return uuidVal
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
