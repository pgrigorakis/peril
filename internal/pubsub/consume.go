package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {

	subCh, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	subQueue, err := subCh.QueueDeclare(
		queueName,
		queueType == SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		false,
		amqp.Table{
			"x-dead-letter-exchange": "peril_dlx",
		},
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = subCh.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return subCh, subQueue, nil

}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		jsonUnmarshaller[T])
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		gobUnmarshaller[T])
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("could not bind queue to exchange: %w", err)
	}

	err = ch.Qos(10, 0, false)
	if err != nil {
		return fmt.Errorf("could not set prefetch size: %w", err)
	}
	deliveryCh, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("could not create channel: %w", err)
	}

	go func() {
		for msg := range deliveryCh {
			decodedMsg, err := unmarshaller(msg.Body)
			if err != nil {
				log.Printf("could not decode message: %v", err)
			}
			ackType := handler(decodedMsg)
			switch ackType {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}
		}
	}()
	return nil
}

func jsonUnmarshaller[T any](msg []byte) (T, error) {
	var dat T
	err := json.Unmarshal(msg, &dat)
	if err != nil {
		return dat, fmt.Errorf("error: %v", err)
	}
	return dat, err
}

func gobUnmarshaller[T any](msg []byte) (T, error) {
	dat := bytes.NewBuffer(msg)
	dec := gob.NewDecoder(dat)

	var decodedMessage T

	err := dec.Decode(&decodedMessage)
	if err != nil {
		return decodedMessage, fmt.Errorf("could not decode value: %v", err)
	}
	return decodedMessage, err
}
