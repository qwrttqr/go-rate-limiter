package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type TokenPayload struct {
	UserID int `json:"user_id"`
}

func GetUserId(token string) (int, error) {
	payloadDecoded := strings.Split(token, ".")[1]
	decoded, err := base64.URLEncoding.DecodeString(payloadDecoded)
	if err != nil {
		log.Fatalf("Decode error: %v", err)
	}
	var payloadJson TokenPayload
	err = json.Unmarshal(decoded, &payloadJson)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return payloadJson.UserID, nil
}
