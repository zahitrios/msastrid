package queues

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type Publisher struct {
	rabbitService
}

func NewPublisher() (*Publisher, error) {
	q := &Publisher{}
	err := q.Create()

	return q, err
}

func (p *Publisher) PublishSkus(skus []interface{}) {
	var consumers []RabbitModel

	jsonAsString := []byte(os.Getenv("RABBIT_CONSUMERS"))
	json.Unmarshal(jsonAsString, &consumers)

	pWg := &sync.WaitGroup{}

	for _, consumer := range consumers {
		pWg.Add(1)
		go p.publishSkus(pWg, consumer, skus)
	}

	pWg.Wait()
}

func (p *Publisher) publishSkus(pWg *sync.WaitGroup, rabbitModel RabbitModel, skus []interface{}) {
	wg := &sync.WaitGroup{}

	for _, sku := range skus {
		body, _ := json.Marshal(sku)
		rabbitModel.Body = string(body)
		wg.Add(1)
		go p.publishSku(wg, rabbitModel)
	}

	wg.Wait()
	pWg.Done()
}

func (p *Publisher) publishSku(wg *sync.WaitGroup, rabbitModel RabbitModel) {
	if err := p.Publish(rabbitModel); err != nil {
		log.Println("publishSku: ", err)
	}

	wg.Done()
}
