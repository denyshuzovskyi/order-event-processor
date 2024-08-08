package model

import "time"

type OrderStatus string

const (
	StatusInitial                OrderStatus = "initial"
	StatusCoolOrderCreated       OrderStatus = "cool_order_created"
	StatusSbuVerificationPending OrderStatus = "sbu_verification_pending"
	StatusConfirmedByMayor       OrderStatus = "confirmed_by_mayor"
	StatusChangedMyMind          OrderStatus = "changed_my_mind"
	StatusFailed                 OrderStatus = "failed"
	StatusChinazes               OrderStatus = "chinazes"
	StatusGiveMyMoneyBack        OrderStatus = "give_my_money_back"
)

var StatusToIsFinal = map[OrderStatus]bool{
	StatusCoolOrderCreated:       false,
	StatusSbuVerificationPending: false,
	StatusConfirmedByMayor:       false,
	StatusChangedMyMind:          true,
	StatusFailed:                 true,
	StatusChinazes:               false,
	StatusGiveMyMoneyBack:        true,
}

type Order struct {
	OrderID     string      `json:"order_id" validate:"required"`
	UserID      string      `json:"user_id" validate:"required"`
	OrderStatus OrderStatus `json:"order_status" validate:"required"`
	IsFinal     bool        `json:"-"`
	UpdatedAt   time.Time   `json:"updated_at" validate:"required"`
	CreatedAt   time.Time   `json:"created_at" validate:"required"`
}

type OrderEvent struct {
	EventID string `json:"event_id" validate:"required"`
	Order
	InOrder bool
}
