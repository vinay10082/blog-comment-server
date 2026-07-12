package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Comment struct {
	ID      int    `json:"id"`
	PostID  int    `json:"postId"`
	UserID  int    `json:"userId"`
	Content string `json:"content"`
}

var db *sql.DB
var jwtSecret = []byte("blog-secret-key-must-be-very-long-and-secure-at-least-256-bits")

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it. Continuing with environment variables.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://blog_user:blog_password@localhost:5432/blog_db?sslmode=disable"
		log.Println("WARNING: DATABASE_URL is not set, using default for local dev")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()
	fmt.Println("Successfully connected to the database!")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("could not create postgres driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "postgres", driver)
	if err != nil {
		log.Fatalf("could not create migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("could not run migrate up: %v", err)
	}
	log.Println("Migrations applied successfully!")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/comments", handleComments)

	handler := corsMiddleware(mux)

	addr := ":" + port
	fmt.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:4200"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Simple check for multiple origins if needed, otherwise just set it directly
		if strings.Contains(allowedOrigins, origin) || allowedOrigins == "*" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// Fallback to exactly what's configured
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractUserIdFromToken(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, fmt.Errorf("missing or invalid authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid claims")
	}

	userIdFloat, ok := claims["userId"].(float64)
	if !ok {
		return 0, fmt.Errorf("userId not found in token")
	}

	return int(userIdFloat), nil
}

func handleComments(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		postId := r.URL.Query().Get("postId")
		if postId == "" {
			http.Error(w, "postId is required", http.StatusBadRequest)
			return
		}

		rows, err := db.Query("SELECT id, post_id, user_id, content FROM sample_comment WHERE post_id = $1", postId)
		if err != nil {
			http.Error(w, "Error fetching comments", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var comments []Comment
		for rows.Next() {
			var c Comment
			if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content); err != nil {
				log.Println("Scan error:", err)
				continue
			}
			comments = append(comments, c)
		}

		if comments == nil {
			comments = make([]Comment, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)

	} else if r.Method == http.MethodPost {
		userId, err := extractUserIdFromToken(r)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		var c Comment
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		c.UserID = userId

		err = db.QueryRow("INSERT INTO sample_comment (post_id, user_id, content) VALUES ($1, $2, $3) RETURNING id",
			c.PostID, c.UserID, c.Content).Scan(&c.ID)
		if err != nil {
			log.Println("Insert error:", err)
			http.Error(w, "Error creating comment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(c)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
