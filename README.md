# QR Access - Go Library

A simple and secure Go library for generating and verifying QR-based access tokens. Perfect for authentication, authorization, and secure access control in your applications.

## Features

- **Secure Token Generation**: HMAC-SHA256 signed tokens ensure tamper-proof QR codes
- **Single-Use Tokens**: Tokens are automatically marked as used after verification
- **Time-Limited Tokens**: Configurable TTL (Time To Live) for token expiration
- **Rate Limiting**: Optional per-user rate limiting to prevent abuse
- **Flexible Storage**: Support for in-memory or Redis storage
- **Easy Configuration**: Simple environment variable-based setup
- **Framework Agnostic**: Pure Go library, works with any framework

## Installation

```bash
go get github.com/tunaaoguzhann/qr-access
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/tunaaoguzhann/qr-access/core"
)

func main() {
    secretKey := "your-secret-key-here"
    
    manager, err := core.NewManager()
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    token, payload, err := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("QR Payload: %s\n", payload)
    
    verifiedToken, err := manager.Verify(ctx, secretKey, payload)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User ID: %s\n", verifiedToken.UserID)
    fmt.Printf("Action: %s\n", verifiedToken.Action)
}
```

### Secret Key Usage

You provide your secret key in both `Generate` and `Verify` functions. The key is used to sign tokens during generation and verify signatures during verification. You can get the key from your environment variables, configuration file, or any other source:

```go
import (
    "os"
    "github.com/tunaaoguzhann/qr-access/core"
)

secretKey := os.Getenv("QR_HMAC_SECRET")
manager, _ := core.NewManager()

token, payload, _ := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
verifiedToken, _ := manager.Verify(ctx, secretKey, payload)
```

The secret key is never stored in the QR payload. It is only used to create and verify the signature.

## Configuration

### Creating a Manager

**Basic:**
```go
manager, err := core.NewManager()
```

**With Options:**
```go
opts := core.ManagerOptions{
    RedisAddr:      "localhost:6379",
    RedisKeyPrefix: "qr-token:",
    MinTTL:         10 * time.Second,
    MaxTTL:         1 * time.Hour,
    RateLimit:      100,
    RateWindow:     1 * time.Hour,
}
manager, err := core.NewManagerWithOptions(opts)
```

### Manager Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `RedisAddr` | string | Redis address (e.g., `localhost:6379`) | Uses in-memory store |
| `RedisKeyPrefix` | string | Redis key prefix | `qr-token:` |
| `MinTTL` | time.Duration | Minimum token TTL | Unlimited |
| `MaxTTL` | time.Duration | Maximum token TTL | Unlimited |
| `RateLimit` | int | Maximum tokens per user per time window | Unlimited |
| `RateWindow` | time.Duration | Rate limit time window | `1 hour` (if RateLimit set) |

### Getting Your Secret Key

The library does not read environment variables. You provide the secret key in both `Generate` and `Verify` functions. Common approaches:

**From environment variable:**
```go
secretKey := os.Getenv("QR_HMAC_SECRET")
manager, _ := core.NewManager()
token, payload, _ := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
verifiedToken, _ := manager.Verify(ctx, secretKey, payload)
```

**From .env file (using godotenv):**
```go
import "github.com/joho/godotenv"

godotenv.Load()
secretKey := os.Getenv("QR_HMAC_SECRET")
manager, _ := core.NewManager()
token, payload, _ := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
```

**Direct value:**
```go
secretKey := "my-secret-key-12345"
manager, _ := core.NewManager()
token, payload, _ := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
```

**From configuration:**
```go
type Config struct {
    QRSecretKey string
}
cfg := loadConfig()
manager, _ := core.NewManager()
token, payload, _ := manager.Generate(ctx, cfg.QRSecretKey, "user-123", "login", 5*time.Minute)
```

## API Reference

### Generate Token

Creates a new QR token and returns the token object and payload string.

```go
token, payload, err := manager.Generate(ctx, secretKey, userID, action, ttl)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `secretKey`: Secret key for HMAC signing (string)
- `userID`: Unique identifier for the user (string)
- `action`: Purpose of the token (e.g., "login", "payment", "access")
- `ttl`: Token validity duration (e.g., `5*time.Minute`)

**Returns:**
- `token`: Token object with ID, UserID, Action, ExpiresAt, and Used fields
- `payload`: Base64-encoded string to embed in QR code
- `err`: Error if generation fails (e.g., rate limit exceeded)

**Example:**
```go
secretKey := "my-secret-key"
token, payload, err := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
if err == core.ErrRateLimitExceeded {
    fmt.Println("Rate limit exceeded")
}
```

### Verify Token

Validates a QR token payload and returns the token information.

```go
verifiedToken, err := manager.Verify(ctx, secretKey, payload)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `secretKey`: Secret key used during token generation (string)
- `payload`: Base64-encoded payload from QR code

**Returns:**
- `verifiedToken`: Token object with all information
- `err`: Error if verification fails

**Possible Errors:**
- `ErrNotFound`: Token not found in store
- `ErrExpired`: Token has expired
- `ErrUsed`: Token has already been used
- `ErrBadSignature`: Signature verification failed (wrong secret key)
- `ErrBadPayload`: Invalid payload format

**Example:**
```go
secretKey := "my-secret-key"
verifiedToken, err := manager.Verify(ctx, secretKey, payload)
if err != nil {
    switch err {
    case core.ErrExpired:
        fmt.Println("Token expired")
    case core.ErrUsed:
        fmt.Println("Token already used")
    case core.ErrBadSignature:
        fmt.Println("Invalid secret key")
    default:
        fmt.Println("Verification failed:", err)
    }
    return
}

fmt.Printf("Verified: User %s, Action %s\n", verifiedToken.UserID, verifiedToken.Action)
```

**Important:** The `secretKey` used in `Verify` must be the same as the one used in `Generate`. If different keys are used, verification will fail with `ErrBadSignature`.

## Storage Options

### In-Memory Store (Default)

Uses in-memory storage. Tokens are lost when the application restarts. Suitable for:
- Development and testing
- Single-server deployments
- Low-traffic applications

```bash
# No REDIS_ADDR needed
export QR_HMAC_SECRET=my-secret
```

### Redis Store

Uses Redis for persistent storage. Tokens survive application restarts. Suitable for:
- Production environments
- Multi-server deployments
- High-traffic applications

```bash
export QR_HMAC_SECRET=my-secret
export REDIS_ADDR=localhost:6379
```

## Rate Limiting

Rate limiting prevents abuse by limiting the number of tokens a user can generate within a time window.

### Enable Rate Limiting

```bash
export QR_RATE_LIMIT_PER_HOUR=100
export QR_RATE_LIMIT_WINDOW_HOURS=1
```

This allows each user to generate up to 100 tokens per hour.

### Rate Limit Behavior

- **With Redis**: Rate limits are shared across all server instances
- **Without Redis**: Rate limits are per-server instance
- **Error**: Returns `ErrRateLimitExceeded` when limit is reached

### Example

```go
secretKey := "my-secret-key"
token, payload, err := manager.Generate(ctx, secretKey, "user-123", "login", 5*time.Minute)
if err == core.ErrRateLimitExceeded {
    fmt.Println("Too many tokens generated. Please try again later.")
    return
}
```

## Advanced Usage

### Using ManagerOptions

For more control, use `NewManagerWithOptions`:

```go
opts := core.ManagerOptions{
    RedisAddr:      "localhost:6379",
    RedisKeyPrefix: "custom-prefix:",
    MinTTL:         10 * time.Second,
    MaxTTL:         1 * time.Hour,
    RateLimit:      50,
    RateWindow:     1 * time.Hour,
}
manager, err := core.NewManagerWithOptions("my-secret-key", opts)
```

## Security Considerations

1. **Secret Key**: 
   - Keep your secret key secure and never commit it to version control
   - Use strong, randomly generated secret keys in production
   - Store secret keys in environment variables or secure configuration management
2. **Token Expiration**: Set appropriate TTL values based on your use case
3. **Rate Limiting**: Enable rate limiting in production to prevent abuse
4. **HTTPS**: Always use HTTPS when transmitting QR payloads
5. **Token Storage**: Use Redis in production for persistent and shared storage

## How It Works

1. **Token Generation**:
   - A unique UUID is generated for the token
   - Token metadata (userID, action, expiration) is stored
   - HMAC signature is created using the secret key
   - Payload (UUID + signature) is base64-encoded for QR embedding

2. **Token Verification**:
   - Payload is decoded to extract UUID and signature
   - Signature is verified using the secret key
   - Token is retrieved from store
   - Expiration and usage status are checked
   - Token is marked as used (single-use enforcement)

## Examples

See the `example/` directory for complete working examples.

## License

This project is open source and available under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
