package token

import (
	"errors"
	"time"
)

var (
	ErrExpiredToken = errors.New("token has expired")
	ErrInvalidToken = errors.New("token is invalid")
)

type Maker interface {
	CreateToken(userId string) (string, error)
	VerifyToken(token string) (*Payload, error)
}

type Payload struct {
	// You can add fields to the payload as needed
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

func NewPayload(userID string, expiresAt time.Time, issuedAt time.Time) *Payload {

	return &Payload{
		UserID:    userID,
		ExpiresAt: expiresAt,
		IssuedAt:  issuedAt,
	}
}

func (p *Payload) Valid() error {
	if time.Now().After(p.ExpiresAt) {
		return ErrExpiredToken
	}
	return nil
}
