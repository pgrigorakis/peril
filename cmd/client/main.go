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
		log.Fatal("could not establish connection: %w", err)
	}

	publishCh, err := connection.Channel()
	if err != nil {
		log.Fatal("could not create channel: %w", err)
	}

	fmt.Println("Connection successful. Starting Peril server...")

	username, err := gamelogic.ClientWelcome()
	if err != nil || username == "" {
		log.Fatal("could not set username or username empty: %w", err)
	}

	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username, //pause.<username>
		routing.PauseKey,              // pause
		pubsub.SimpleQueueTransient,
		handlerPause(gameState))
	if err != nil {
		log.Fatal("could not declare and bind queue: %w", err)
	}

	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username, //army_moves.<username>,
		routing.ArmyMovesPrefix+".*",         //army_moves.*
		pubsub.SimpleQueueTransient,
		handlerMove(gameState, publishCh))
	if err != nil {
		log.Fatal("could not declare and bind queue: %w", err)
	}

	err = pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueDurable,
		handlerWar(gameState, publishCh))
	if err != nil {
		log.Fatal("could not declare and bind queue: %w", err)
	}

	for {
		playerInput := gamelogic.GetInput()
		if len(playerInput) == 0 {
			log.Println("Usage: <command> <unit type|location>")
			continue
		}

		switch playerInput[0] {
		case "spawn":
			err := gameState.CommandSpawn(playerInput)
			if err != nil {
				fmt.Printf("could not publish message: %v\n", err)
				continue
			}
		case "move":
			move, err := gameState.CommandMove(playerInput)
			if err != nil {
				fmt.Printf("could not publish message: %v\n", err)
			} else {
				err = pubsub.PublishJSON(
					publishCh,
					routing.ExchangePerilTopic,
					routing.ArmyMovesPrefix+"."+username,
					move)
				fmt.Println("Move successful.")
			}
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("I don't understand this command.")
		}
	}
}
