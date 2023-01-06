package brokers

import (
	"fmt"
	"sync"

	zlog "github.com/rs/zerolog/log"
)

var SecretBox = bSecretBox{
	scrts: make(map[SecretId]Secret),
}

// Secret Id
type SecretId string

type Secret interface {
	GetSecretId() SecretId
}

// Empty Secret
type EmptySecret struct{}

func (*EmptySecret) GetSecretId() SecretId {
	return "empty secret"
}

// Not Found Error
type SecretNotFoundError struct {
	id SecretId
}

func (s *SecretNotFoundError) Error() string {
	return fmt.Sprintf("secret with id: %s not found", s.id)
}

// Wrong Secret Error
type WrongSecretError struct {
	id   SecretId
	desc string
}

func (s *WrongSecretError) Error() string {
	return fmt.Sprintf("wrong secret with id: %s provided, error description: %s", s.id, s.desc)
}

// Secret Box
type bSecretBox struct {
	scrts map[SecretId]Secret
	lock  sync.RWMutex
}

func (sb *bSecretBox) GetSecret(id SecretId) (Secret, error) {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	secret, ok := sb.scrts[id]
	if ok {
		return secret, nil
	}

	// fetch db

	secret, ok = sb.scrts[id]
	if ok {
		return secret, nil
	}
	return &EmptySecret{}, &SecretNotFoundError{id: id}
}

func (sb *bSecretBox) SetSecret(s Secret) (SecretId, error) {
	sb.lock.RLock()
	defer sb.lock.RUnlock()

	sb.scrts[s.GetSecretId()] = s

	// update db

	zlog.Info().Msgf("successfully setup secret: %s", s.GetSecretId())
	return s.GetSecretId(), nil
}
