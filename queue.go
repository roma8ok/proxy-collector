package main

import (
	"github.com/streadway/amqp"
)

func publish(ch *amqp.Channel, queueName string, body []byte) error {
	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
	if err != nil {
		return err
	}

	return nil
}

func consume(ch *amqp.Channel, queueName string, handler func([]byte) error) error {
	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for m := range msgs {
		if err := handler(m.Body); err != nil {
			if err := m.Nack(false, false); err != nil {
				return err
			}
		} else {
			if err := m.Ack(false); err != nil {
				return err
			}
		}
	}

	return nil
}
