package model

import "time"

type Order struct {
	Number     string
	UserID     int64
	Status     string
	Accrual    *float64
	UploadedAt time.Time
}

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)
