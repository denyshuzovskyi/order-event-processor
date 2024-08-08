package broadcaster

import (
	"order-event-processor/internal/model"
)

type FromDbEventProducerStorage interface {
	RunInTransaction(f func()) error
	AcquireLock(id string) error
	GetAllEventsByOrderId(orderId string) ([]model.OrderEvent, error)
}

type FromDbEventProducer struct {
	repository            FromDbEventProducerStorage
	OrderEventBroadcaster *OrderEventBroadcaster
}

func NewFromDbEventProducer(repository FromDbEventProducerStorage) *FromDbEventProducer {
	return &FromDbEventProducer{
		repository: repository,
	}
}

func (p *FromDbEventProducer) OnRegistration(orderId string, channel chan *model.OrderEvent) {
	go p.produce(orderId, channel)
}

func (p *FromDbEventProducer) produce(orderId string, channel chan *model.OrderEvent) {
	p.repository.RunInTransaction(func() {
		p.repository.AcquireLock(orderId)

		events, err := p.repository.GetAllEventsByOrderId(orderId)
		if err != nil {
			p.OrderEventBroadcaster.UnregisterChannel(orderId, channel)
			// todo log
		}
		for _, event := range events {
			p.OrderEventBroadcaster.Broadcast(&event)
		}
	})
}
