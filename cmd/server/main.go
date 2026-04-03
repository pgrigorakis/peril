package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connURL := "amqp://guest:guest@localhost:5672/"

	connection, err := amqp.Dial(connURL)
	if err != nil {
		log.Fatal("could not establish connection to RabbitMQ: %w", err)
	}
	defer connection.Close()
	fmt.Println("Connected to RabbitMQ.")

	publishCh, err := connection.Channel()
	if err != nil {
		log.Fatal("could not create channel: %w", err)
	}

	err = pubsub.SubscribeGob(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
		handlerLogs())
	if err != nil {
		log.Fatal("could not declare and bind queue: %w", err)
	}

	gamelogic.PrintServerHelp()

	for {
		playerInput := gamelogic.GetInput()
		if len(playerInput) == 0 {
			continue
		}
		switch playerInput[0] {
		case "pause":
			log.Println("Printing pause message.")
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Printf("could not publish message: %v\n", err)
			}
		case "resume":
			log.Println("Printing resume message.")
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Printf("could not publish message: %v\n", err)
			}
		case "quit":
			log.Println("Quitting.")
			return
		default:
			log.Println("I don't understand this command.")
		}
	}
}
