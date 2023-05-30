package queues

import (
	"context"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitService struct {
	conn amqp.Connection
	ch   amqp.Channel
}

type RabbitModel struct {
	Exchange string `json:"exchange"`
	Name     string `json:"routing_key"`
	Body     string
}

func NewRabbitService() *rabbitService {
	return &rabbitService{}
}

func (r *rabbitService) Create() error {
	port, _ := strconv.Atoi(os.Getenv("RABBIT_PORT"))

	url := amqp.URI{
		Scheme:   "amqp",
		Host:     os.Getenv("RABBIT_HOST"),
		Port:     port,
		Username: os.Getenv("RABBIT_USER"),
		Password: os.Getenv("RABBIT_PASS"),
		Vhost:    os.Getenv("RABBIT_VHOST"),
	}.String()

	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	r.conn = *conn

	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	r.ch = *ch

	return nil
}

func (r *rabbitService) Publish(rabbitModel RabbitModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), (5 * time.Second))
	defer cancel()

	err := r.ch.PublishWithContext(ctx,
		rabbitModel.Exchange, // exchange
		rabbitModel.Name,     // routing key
		false,                // mandatory
		false,                // immediate
		amqp.Publishing{
			DeliveryMode: 2,
			Body:         []byte(rabbitModel.Body),
		})

	return err
}
