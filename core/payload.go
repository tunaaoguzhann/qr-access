package core

import (
	"encoding/base64"
	"encoding/json"
)

type payload struct {
	ID  string `json:"id"`
	Sig string `json:"sig"`
}

func EncodePayload(id, signature string) (string, error) {
	data := payload{ID: id, Sig: signature}
	raw, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DecodePayload(encoded string) (payload, error) {
	var data payload
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return data, ErrBadPayload
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, ErrBadPayload
	}
	return data, nil
}

