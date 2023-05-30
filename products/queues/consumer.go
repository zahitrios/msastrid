package queues

import "log"

type Consumer struct {
	rabbitService
}

func NewConsumer() (*Consumer, error) {
	q := &Consumer{}
	err := q.Create()

	return q, err
}

func (c *Consumer) Receive(queue string, callback func(body []byte)) error {
	if err := c.ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		log.Println("err setting qos: ", err)
	}

	msgs, err := c.ch.Consume(
		queue, // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		return err
	}

	var forever chan struct{}

	go func() {
		for msg := range msgs {
			callback(msg.Body)
		}
	}()

	<-forever

	return nil
}
