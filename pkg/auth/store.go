package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/zalando/go-keyring"
)

const service = "corectl"

type Token struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	TokenType    string    `json:"tokenType,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
}

type Store interface {
	Get(origin string) (Token, error)
	Set(origin string, token Token) error
	Delete(origin string) error
}

type KeyringStore struct{}

func account(origin string) string {
	sum := sha256.Sum256([]byte(origin))
	return "portal:" + hex.EncodeToString(sum[:])
}

func (KeyringStore) Get(origin string) (Token, error) {
	raw, err := keyring.Get(service, account(origin))
	if err != nil {
		return Token{}, err
	}
	var token Token
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return Token{}, err
	}
	return token, nil
}

func (KeyringStore) Set(origin string, token Token) error {
	b, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return keyring.Set(service, account(origin), string(b))
}

func (KeyringStore) Delete(origin string) error {
	err := keyring.Delete(service, account(origin))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
