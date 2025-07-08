package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"simple-butler-server/models"
	"strings"
)

// AuthMiddleware handles API key authentication
func AuthMiddleware(db models.Database) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from Authorization header or query parameter
			apiKey := extractAPIKey(r)
			fmt.Printf("Auth middleware - API key received: '%s'\n", apiKey)
			fmt.Printf("Auth middleware - Request URL: %s\n", r.URL.String())
			fmt.Printf("Auth middleware - Headers: %v\n", r.Header)

			if apiKey == "" {
				fmt.Printf("Auth middleware - No API key found\n")
				http.Error(w, `{"errors":["missing api_key"]}`, http.StatusUnauthorized)
				return
			}

			// Look up user by API key
			user, err := db.GetUserByAPIKey(apiKey)
			if err != nil {
				fmt.Printf("Auth middleware - API key lookup failed: %v\n", err)
				http.Error(w, `{"errors":["invalid api_key"]}`, http.StatusUnauthorized)
				return
			}

			fmt.Printf("Auth middleware - Found user: %s (ID: %d)\n", user.Username, user.ID)

			// Add user to request context
			ctx := r.Context()
			ctx = SetUser(ctx, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware handles optional authentication (for public endpoints that can be enhanced with auth)
func OptionalAuthMiddleware(db models.Database) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from Authorization header or query parameter
			apiKey := extractAPIKey(r)
			if apiKey != "" {
				// Look up user by API key if provided
				user, err := db.GetUserByAPIKey(apiKey)
				if err == nil {
					// Add user to request context
					ctx := r.Context()
					ctx = SetUser(ctx, user)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractAPIKey extracts API key from request
func extractAPIKey(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	fmt.Printf("Authorization header: '%s'\n", authHeader)
	if authHeader != "" {
		fmt.Printf("Using Authorization header\n")
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// Remove "access_token=" prefix if present (butler sends this format)
		if strings.HasPrefix(authHeader, "access_token=") {
			return strings.TrimPrefix(authHeader, "access_token=")
		}
		return authHeader
	}

	// Try query parameter
	apiKey := r.URL.Query().Get("api_key")
	fmt.Printf("Raw query parameter api_key: '%s'\n", apiKey)

	// Handle butler's format: access_token=<actual_token>
	if strings.HasPrefix(apiKey, "access_token=") {
		parsed := strings.TrimPrefix(apiKey, "access_token=")
		fmt.Printf("Parsed API key from access_token format: '%s'\n", parsed)
		return parsed
	}

	fmt.Printf("Returning raw query parameter: '%s'\n", apiKey)
	return apiKey
}

// GenerateAPIKey generates a new API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateTestUser creates a test user for development
func CreateTestUser(db models.Database, username string) (*models.User, error) {
	apiKey, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:    username,
		DisplayName: username,
		APIKey:      apiKey,
	}

	err = db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Created test user: %s with API key: %s\n", username, apiKey)
	return user, nil
}
