package model

import "time"

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateUserInput struct {
	Email    string
	Password string
}

type UpdateUserInput struct {
	Email    *string
	Password *string
}
