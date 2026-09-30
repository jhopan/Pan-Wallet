package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
)

var ErrUnauthorized = errors.New("product API key tidak valid")

type ProductKeyStore interface {
	ProductForTokenHash(context.Context, []byte) (string, error)
}

type ProductAuthenticator struct {
	store ProductKeyStore
}

func NewProductAuthenticator(store ProductKeyStore) ProductAuthenticator {
	return ProductAuthenticator{store: store}
}

func (authenticator ProductAuthenticator) Authenticate(ctx context.Context, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(token))
	product, err := authenticator.store.ProductForTokenHash(ctx, hash[:])
	if err != nil {
		return "", ErrUnauthorized
	}
	return product, nil
}
