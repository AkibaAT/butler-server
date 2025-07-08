package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"simple-butler-server/auth"
	"simple-butler-server/handlers"
	"simple-butler-server/models"
	"simple-butler-server/storage"

	"github.com/gorilla/mux"
)

func main() {
	// Command line flags
	var (
		port           = flag.String("port", "8080", "Port to run the server on")
		dbPath         = flag.String("db", "./butler-server.db", "Path to SQLite database")
		storagePath    = flag.String("storage", "./storage", "Path to file storage directory")
		createUser     = flag.String("create-user", "", "Create a regular user with the given username")
		createAdmin    = flag.String("create-admin", "", "Create an admin user with the given username")
		listUsers      = flag.Bool("list-users", false, "List all users in the database")
		deactivateUser = flag.String("deactivate-user", "", "Deactivate user with the given username")
		activateUser   = flag.String("activate-user", "", "Activate user with the given username")
	)
	flag.Parse()

	// Initialize database
	db, err := models.NewSQLiteDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = db.Migrate()
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize storage
	localStorage, err := storage.NewLocalStorage(*storagePath, db)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Handle user management commands
	if *createUser != "" {
		_, err := auth.CreateUser(db, *createUser, "user")
		if err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		os.Exit(0)
	}

	if *createAdmin != "" {
		_, err := auth.CreateUser(db, *createAdmin, "admin")
		if err != nil {
			log.Fatalf("Failed to create admin: %v", err)
		}
		os.Exit(0)
	}

	if *listUsers {
		err := auth.ListUsers(db)
		if err != nil {
			log.Fatalf("Failed to list users: %v", err)
		}
		os.Exit(0)
	}

	if *deactivateUser != "" {
		err := auth.DeactivateUser(db, *deactivateUser)
		if err != nil {
			log.Fatalf("Failed to deactivate user: %v", err)
		}
		os.Exit(0)
	}

	if *activateUser != "" {
		err := auth.ActivateUser(db, *activateUser)
		if err != nil {
			log.Fatalf("Failed to activate user: %v", err)
		}
		os.Exit(0)
	}

	// Initialize handlers
	coreHandlers := handlers.NewCoreHandlers(db)
	wharfHandlers := handlers.NewWharfHandlers(db, localStorage)

	// Setup router
	r := mux.NewRouter()

	// Add CORS middleware for development and request logging
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Log all requests for debugging
			fmt.Printf("REQUEST: %s %s\n", req.Method, req.URL.String())

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if req.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, req)
		})
	})

	// Public routes (no authentication required)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"Simple Butler Server","version":"1.0.0"}`)
	}).Methods("GET")

	// OAuth endpoints for butler login
	oauthHandler := func(w http.ResponseWriter, r *http.Request) {
		// Log the request for debugging
		fmt.Printf("OAuth request: %s %s\n", r.Method, r.URL.String())
		fmt.Printf("Query params: %v\n", r.URL.Query())

		clientID := r.URL.Query().Get("client_id")
		if clientID != "butler" {
			http.Error(w, fmt.Sprintf("Invalid client_id: %s", clientID), http.StatusBadRequest)
			return
		}

		// Extract redirect_uri to get the port
		redirectURI := r.URL.Query().Get("redirect_uri")
		if redirectURI == "" {
			http.Error(w, "Missing redirect_uri", http.StatusBadRequest)
			return
		}

		// For development, create a simple test user or get existing one
		user, err := auth.CreateTestUser(db, "testuser")
		if err != nil {
			// User might already exist, try to get existing user
			fmt.Printf("User already exists, looking up existing user...\n")

			// Try to find existing testuser
			existingUser, lookupErr := db.GetUserByID(1) // Assume first user is testuser
			if lookupErr != nil {
				// If we can't find the user, fall back to a known API key
				fmt.Printf("Could not find existing user, using fallback API key\n")
				redirectURL := redirectURI + "#access_token=test-api-key-12345"
				http.Redirect(w, r, redirectURL, http.StatusFound)
				return
			}
			user = existingUser
		}

		// Redirect back to butler with API key
		redirectURL := redirectURI + "#access_token=" + user.APIKey
		fmt.Printf("Redirecting to: %s\n", redirectURL)
		fmt.Printf("API key being sent: %s\n", user.APIKey)

		// Instead of redirect, let's show a page with the redirect info
		w.Header().Set("Content-Type", "text/html")
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>Butler Login</title></head>
<body>
<h1>Butler Login Successful</h1>
<p>API Key: <code>%s</code></p>
<p>Redirecting to butler...</p>
<script>
window.location.href = "%s";
</script>
</body>
</html>`, user.APIKey, redirectURL)
		w.Write([]byte(html))
	}

	// Register OAuth handler for both paths butler might use
	r.HandleFunc("/oauth/authorize", oauthHandler).Methods("GET")
	r.HandleFunc("/user/oauth", oauthHandler).Methods("GET")

	// API routes with optional authentication
	api := r.PathPrefix("/").Subrouter()
	api.Use(auth.OptionalAuthMiddleware(db))

	// Core API endpoints
	api.HandleFunc("/profile", coreHandlers.GetProfile).Methods("GET")
	api.HandleFunc("/profile/games", coreHandlers.GetProfileGames).Methods("GET")
	api.HandleFunc("/games/{id}", coreHandlers.GetGame).Methods("GET")
	api.HandleFunc("/games/{id}/uploads", coreHandlers.GetGameUploads).Methods("GET")
	api.HandleFunc("/uploads/{id}", coreHandlers.GetUpload).Methods("GET")
	api.HandleFunc("/uploads/{id}/builds", coreHandlers.GetUploadBuilds).Methods("GET")
	api.HandleFunc("/uploads/{id}/download", coreHandlers.GetUploadDownload).Methods("GET")
	api.HandleFunc("/builds/{id}", coreHandlers.GetBuild).Methods("GET")

	// Wharf API endpoints
	wharf := r.PathPrefix("/wharf").Subrouter()
	wharf.Use(auth.AuthMiddleware(db))

	wharf.HandleFunc("/status", wharfHandlers.GetWharfStatus).Methods("GET")
	wharf.HandleFunc("/channels", wharfHandlers.ListChannels).Methods("GET")
	wharf.HandleFunc("/channels/{channel}", wharfHandlers.GetChannel).Methods("GET")
	wharf.HandleFunc("/builds", wharfHandlers.CreateBuild).Methods("POST")
	wharf.HandleFunc("/builds/{id}/files", wharfHandlers.GetBuildFiles).Methods("GET")
	wharf.HandleFunc("/builds/{id}/files", wharfHandlers.CreateBuildFile).Methods("POST")
	wharf.HandleFunc("/builds/{buildId}/files/{fileId}", wharfHandlers.FinalizeBuildFile).Methods("POST")
	wharf.HandleFunc("/builds/{buildId}/files/{fileId}/download", wharfHandlers.GetBuildFileDownload).Methods("GET", "HEAD")

	// Upload endpoints
	upload := r.PathPrefix("/upload").Subrouter()
	upload.HandleFunc("/{sessionId}", localStorage.HandleUpload).Methods("POST", "PUT", "PATCH")

	// Download endpoints
	download := r.PathPrefix("/downloads").Subrouter()
	download.HandleFunc("/builds/{buildId}/files/{fileId}", localStorage.HandleDownload).Methods("GET")
	download.HandleFunc("/uploads/{uploadId}/{filename}", localStorage.HandleDownload).Methods("GET")

	// Start server
	fmt.Printf("Starting server on port %s\n", *port)
	fmt.Printf("Database: %s\n", *dbPath)
	fmt.Printf("Storage: %s\n", *storagePath)
	fmt.Printf("\nTo create a test user, run:\n")
	fmt.Printf("  %s -create-user=myusername\n", os.Args[0])
	fmt.Printf("\nThen configure butler with:\n")
	fmt.Printf("  butler --address=http://127.0.0.1:%s login\n", *port)
	fmt.Printf("\nOr add '127.0.0.1 api.localhost' to /etc/hosts and use:\n")
	fmt.Printf("  butler --address=http://localhost:%s login\n", *port)

	address := "0.0.0.0:" + *port
	fmt.Printf("Server listening on %s (all interfaces)\n", address)
	log.Fatal(http.ListenAndServe(address, r))
}
