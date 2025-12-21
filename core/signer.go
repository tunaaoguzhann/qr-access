package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

type Signer struct {
	secret []byte
}

func NewSigner(secret string) *Signer {
	return &Signer{secret: []byte(secret)}
}

func (s *Signer) Sign(idBytes []byte) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(idBytes)
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

func (s *Signer) Verify(idBytes []byte, signature string) bool {
	expected := s.Sign(idBytes)
	return hmac.Equal([]byte(expected), []byte(signature))
}

