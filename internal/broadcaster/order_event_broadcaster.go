package broadcaster

import (
	"order-event-processor/internal/model"
	"slices"
	"sync"
)

type RegistrationListener interface {
	OnRegistration(orderId string, channel chan *model.OrderEvent)
}

type OrderEventBroadcaster struct {
	mutex                sync.Mutex
	orderIdToChannels    map[string][]chan *model.OrderEvent
	registrationListener RegistrationListener
}

func NewOrderEventBroadcaster(listener RegistrationListener) *OrderEventBroadcaster {
	return &OrderEventBroadcaster{
		orderIdToChannels:    make(map[string][]chan *model.OrderEvent),
		registrationListener: listener,
	}
}

func (b *OrderEventBroadcaster) Broadcast(event *model.OrderEvent) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	channels := b.orderIdToChannels[event.OrderID]
	for _, channel := range channels {
		channel <- event
	}
	if event.IsFinal {
		for _, channel := range channels {
			close(channel)
			delete(b.orderIdToChannels, event.OrderID)
		}
	}
}

func (b *OrderEventBroadcaster) UnregisterChannel(orderId string, channel chan *model.OrderEvent) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	close(channel)

	channels, ok := b.orderIdToChannels[orderId]
	if !ok {
		return
	}
	index := slices.Index(channels, channel)
	if index >= 0 {
		b.orderIdToChannels[orderId] = slices.Delete(channels, index, index)
	}
}

func (b *OrderEventBroadcaster) RegisterChannel(orderId string, channel chan *model.OrderEvent) {
	b.addChannelToMap(orderId, channel)
	b.registrationListener.OnRegistration(orderId, channel)
}

func (b *OrderEventBroadcaster) addChannelToMap(orderId string, channel chan *model.OrderEvent) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	channels, ok := b.orderIdToChannels[orderId]
	if !ok {
		b.orderIdToChannels[orderId] = []chan *model.OrderEvent{channel}
	} else {
		b.orderIdToChannels[orderId] = append(channels, channel)
	}
}
