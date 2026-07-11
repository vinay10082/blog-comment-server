package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it. Continuing with environment variables.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("WARNING: DATABASE_URL is not set")
	} else {
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("Error opening database: %v", err)
		}
		defer db.Close()

		if err = db.Ping(); err != nil {
			log.Fatalf("Error connecting to the database: %v", err)
		}
		fmt.Println("Successfully connected to the database!")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Comment Server Running")
	})
	
	addr := ":" + port
	fmt.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
