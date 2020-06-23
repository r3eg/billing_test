package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var channel *amqp.Channel

func initPublisher(amqpURI, exchange, exchangeType string, isInit chan string) error {
	amqpHost := os.Getenv("AMQP_HOST")
	if amqpHost == "" {
		amqpHost = "localhost"
	}
	// This function dials, connects, declares, publishes, and tears down,
	// all in one go. In a real service, you probably want to maintain a
	// long-lived connection as state, and publish against that.

	log.Printf("dialing %q", amqpURI)
	var connection *amqp.Connection
	var err error
	isConnected := false
	for !isConnected {
		connection, err = amqp.Dial(amqpURI)
		if err != nil {
			logrus.Error("error connect to amqp: ", err)
			logrus.Warn("try reconnect after 15 seconds")
			time.Sleep(time.Second * 15)
		}
		isConnected = true
	}
	defer connection.Close()

	log.Printf("got Connection, getting Channel")
	channel, err = connection.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}

	log.Printf("got Channel, declaring %q Exchange (%q)", exchangeType, exchange)
	if err := channel.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("Exchange Declare: %s", err)
	}
	isInit <- "success"
	for {

	}
	return nil
}

func Publish(exchange, routingKey string, body []byte) error {
	if err := channel.Publish(
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Body:            body,
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		return err
	}
	return nil
}
