package model

import "time"

type User struct {
	ID           int64
	Login        string
	PasswordHash []byte
	CreatedAt    time.Time
}
