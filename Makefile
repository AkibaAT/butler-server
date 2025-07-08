.PHONY: build run test clean deps create-user

# Build the server
build:
	go build -o butler-server .

# Run the server in development mode
run: build
	./butler-server

# Run with custom settings
dev: build
	./butler-server -port=8080 -db=./dev.db -storage=./dev-storage

# Install dependencies
deps:
	go mod tidy
	go mod download

# Create a test user
create-user: build
	./butler-server -create-user=testuser

# Test the API
test-api:
	@echo "Testing server health..."
	curl -s http://localhost:8080/ | jq .
	@echo "\nTesting with test user API key..."
	@echo "First create a user with: make create-user"

# Clean build artifacts
clean:
	rm -f butler-server butler-server.exe
	rm -f *.db
	rm -rf storage/ dev-storage/

# Quick setup for development
setup: deps build create-user
	@echo "Setup complete! Run 'make run' to start the server"

# Test butler integration
test-butler:
	@echo "Testing butler integration..."
	@echo "Make sure server is running first!"
	butler --address=http://localhost:8080 login

# Build for different platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o butler-server-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o butler-server-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o butler-server-windows-amd64.exe .

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the server binary"
	@echo "  run        - Build and run the server"
	@echo "  dev        - Run with development settings"
	@echo "  deps       - Install Go dependencies"
	@echo "  create-user- Create a test user"
	@echo "  test-api   - Test the API endpoints"
	@echo "  clean      - Clean build artifacts"
	@echo "  setup      - Quick setup for development"
	@echo "  build-all  - Build for all platforms"
	@echo "  help       - Show this help"