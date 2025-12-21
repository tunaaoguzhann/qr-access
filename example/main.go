package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tunaaoguzhann/qr-access/core"
)

func main() {
	secretKey := "my-secret-key-12345"
	
	manager, err := core.NewManager()
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	userID := "user-123"
	action := "login"
	ttl := 5 * time.Minute

	token, payload, err := manager.Generate(ctx, secretKey, userID, action, ttl)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	fmt.Printf("Generated QR Token:\n")
	fmt.Printf("  Token ID: %s\n", token.ID)
	fmt.Printf("  User ID: %s\n", token.UserID)
	fmt.Printf("  Action: %s\n", token.Action)
	fmt.Printf("  Expires At: %s\n", token.ExpiresAt)
	fmt.Printf("  QR Payload: %s\n", payload)
	fmt.Printf("\nUse this payload in a QR code!\n\n")

	verifiedToken, err := manager.Verify(ctx, secretKey, payload)
	if err != nil {
		log.Fatalf("Failed to verify token: %v", err)
	}

	fmt.Printf("Verified Token:\n")
	fmt.Printf("  Token ID: %s\n", verifiedToken.ID)
	fmt.Printf("  User ID: %s\n", verifiedToken.UserID)
	fmt.Printf("  Action: %s\n", verifiedToken.Action)
	fmt.Printf("  Used: %v\n", verifiedToken.Used)

	_, err = manager.Verify(ctx, secretKey, payload)
	if err != nil {
		fmt.Printf("\nAs expected, token cannot be used twice: %v\n", err)
	}
}

