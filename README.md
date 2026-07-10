# Blog Comment Server

This is the Go microservice responsible for handling comments on blog posts.

## Configuration
Configuration is managed via environment variables. Create a `.env` file at the root to customize settings:
```env
PORT=8082
```

## Running Locally
To run the server locally without Docker:
```bash
go run main.go
```
