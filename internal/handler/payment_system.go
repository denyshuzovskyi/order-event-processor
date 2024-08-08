package handler

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"order-event-processor/internal/broadcaster"
	"order-event-processor/internal/model"
	"slices"
	"sort"
	"time"
)

type OrderEventRepository interface {
	SaveOrderEvent(event model.OrderEvent) error
	InsertOrderEventsOrUpdateIsInOrder(events ...model.OrderEvent) error
	GetAllEventsByOrderId(orderId string) ([]model.OrderEvent, error)
	RunInTransaction(run func()) error
	AcquireLock(id string) error
	ExistsOrderEventForOrderIdFinalAndInOrder(orderId string) (bool, error)
	ExistsOrderEventWithEventId(eventId string) (bool, error)
	ExistsOrderEventForOrderIdWithStatus(orderId string, orderStatus model.OrderStatus) (bool, error)
	UpdateOrderEventFinalStatus(eventId string) error
	InsertOrUpdateOrder(order model.Order) error
}

type PaymentSystemEventHandler struct {
	log         *slog.Logger
	validate    *validator.Validate
	repository  OrderEventRepository
	broadcaster *broadcaster.OrderEventBroadcaster
}

func NewPaymentSystemEventHandler(log *slog.Logger, validate *validator.Validate, repository OrderEventRepository, broadcaster *broadcaster.OrderEventBroadcaster) *PaymentSystemEventHandler {
	return &PaymentSystemEventHandler{
		log:         log,
		validate:    validate,
		repository:  repository,
		broadcaster: broadcaster,
	}
}

func (h *PaymentSystemEventHandler) Handle(w http.ResponseWriter, r *http.Request) {
	paymentSystemEvent := model.OrderEvent{}
	if err := json.NewDecoder(r.Body).Decode(&paymentSystemEvent); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Error("failed to decode Payment system broadcaster", "error", err)
		return
	}

	if err := h.validate.Struct(paymentSystemEvent); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.log.Error("validation error", "error", err)
		return
	}

	h.log.Debug("received payload", "PaymentSystemEvent", paymentSystemEvent)

	h.repository.RunInTransaction(func() {
		h.repository.AcquireLock(paymentSystemEvent.OrderID)

		exists, err := h.repository.ExistsOrderEventWithEventId(paymentSystemEvent.EventID)
		if exists {
			w.WriteHeader(http.StatusConflict)
			h.log.Error("order event was already processed", "error", err)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error("failed to check if order is final and in order", "error", err)
			return
		}

		exists, err = h.repository.ExistsOrderEventForOrderIdFinalAndInOrder(paymentSystemEvent.OrderID)
		if exists {
			w.WriteHeader(http.StatusGone)
			h.log.Error("order is already final and in order", "error", err)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error("failed to check if order is final and in order", "error", err)
			return
		}

		events, err := h.repository.GetAllEventsByOrderId(paymentSystemEvent.OrderID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error("failed to retrieve order events", "error", err)
			return
		}

		newInOrderEvents, err := h.getUpdatedInOrderEvents(append(events, paymentSystemEvent))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			h.log.Error("failed to process broadcaster", "error", err)
			return
		}

		if err := h.repository.InsertOrderEventsOrUpdateIsInOrder(newInOrderEvents...); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error("error while saving broadcaster", "error", err)
			return
		}

		if !containsId(newInOrderEvents, paymentSystemEvent.EventID) {
			if err := h.repository.InsertOrderEventsOrUpdateIsInOrder(paymentSystemEvent); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error("error while saving broadcaster", "error", err)
				return
			}
		}
		if len(newInOrderEvents) > 0 {
			if err := h.repository.InsertOrUpdateOrder(newInOrderEvents[len(newInOrderEvents)-1].Order); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error("error while saving order", "error", err)
				return
			}

			for _, event := range newInOrderEvents {
				h.broadcaster.Broadcast(&event)
			}
		}
	})
}

func containsId(orderEvents []model.OrderEvent, id string) bool {
	for _, event := range orderEvents {
		if event.EventID == id {
			return true
		}
	}
	return false
}

var transitions = map[model.OrderStatus][]model.OrderStatus{
	model.StatusInitial:                {model.StatusCoolOrderCreated},
	model.StatusCoolOrderCreated:       {model.StatusSbuVerificationPending, model.StatusChangedMyMind, model.StatusFailed},
	model.StatusSbuVerificationPending: {model.StatusConfirmedByMayor, model.StatusChangedMyMind, model.StatusFailed},
	model.StatusConfirmedByMayor:       {model.StatusChinazes, model.StatusChangedMyMind, model.StatusFailed},
	model.StatusChinazes:               {model.StatusGiveMyMoneyBack},
}

func (h *PaymentSystemEventHandler) getUpdatedInOrderEvents(orderEvents []model.OrderEvent) ([]model.OrderEvent, error) {
	sort.Slice(orderEvents, func(i, j int) bool {
		return orderEvents[i].UpdatedAt.Before(orderEvents[j].UpdatedAt)
	})
	var changedOrderEvents []model.OrderEvent
	currentStatus := model.StatusInitial
	isFinalized := false
	for _, event := range orderEvents {
		if event.InOrder {
			currentStatus = event.OrderStatus
			continue
		}
		statuses := transitions[currentStatus]
		if isFinalized {
			return nil, fmt.Errorf("invalid broadcaster %s, after final status", event.EventID)
		} else if slices.Contains(statuses, event.OrderStatus) {
			event.InOrder = true
			event.IsFinal = model.StatusToIsFinal[event.OrderStatus]
			isFinalized = event.IsFinal
			changedOrderEvents = append(changedOrderEvents, event)
			currentStatus = event.OrderStatus

			if event.OrderStatus == model.StatusChinazes {
				go func() {
					h.log.Debug("started status update job")
					time.Sleep(30 * time.Second)
					h.repository.RunInTransaction(func() {
						h.repository.AcquireLock(event.OrderID)
						moneyBackStatusExists, err := h.repository.ExistsOrderEventForOrderIdWithStatus(event.OrderID, model.StatusGiveMyMoneyBack)
						if moneyBackStatusExists {
							return
						} else if err != nil {
							h.log.Error("error while checking existence of order with status", "error", err)
							return
						}
						err = h.repository.UpdateOrderEventFinalStatus(event.EventID)
						if err != nil {
							h.log.Error("error while updating event order status", "error", err)
							return
						}
						event.IsFinal = true
						if err := h.repository.InsertOrUpdateOrder(event.Order); err != nil {
							h.log.Error("error while saving order", "error", err)
							return
						}
						h.broadcaster.Broadcast(&event)
					})
					h.log.Debug("update job finished")
				}()
			}
		} else {
			break
		}
	}
	return changedOrderEvents, nil
}

//func processOrderEvents2(orderEvents []model.EventHolder) {
//	statusToEvent := make(map[model.OrderStatus]model.EventHolder)
//	for _, broadcaster := range orderEvents {
//		statusToEvent[broadcaster.OrderStatus] = broadcaster
//	}
//
//	currentEvent := model.StatusInitial
//	index := 1
//	for {
//		statuses := transitions[currentEvent]
//		if broadcaster, ok := searchFirstInMap(statusToEvent, statuses); ok {
//			broadcaster.Index = index
//			broadcaster.IsFinal = model.StatusToIsFinal[broadcaster.OrderStatus]
//			index++
//			currentEvent = broadcaster.OrderStatus
//		} else {
//			break
//		}
//	}
//
//}
//
//func searchFirstInMap(statusToEvent map[model.OrderStatus]model.EventHolder, statuses []model.OrderStatus) (*model.EventHolder, bool) {
//	for _, status := range statuses {
//		if broadcaster, ok := statusToEvent[status]; ok {
//			return &broadcaster, true
//		}
//	}
//	return nil, false
//}

// Define a function to validate the status transition
//func validateTransition(from, to model.OrderStatus) error {
//
//	allowed, exists := transitions[from]
//	if !exists {
//		return errors.New("invalid status")
//	}
//
//	for _, status := range allowed {
//		if status == to {
//			return nil
//		}
//	}
//
//	return errors.New("invalid transition")
//}
