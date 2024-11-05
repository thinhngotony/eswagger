package main

import (
	"encoding/json"
	"fmt"
	"log"
	"main/eswagger"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func createUser(w http.ResponseWriter, r *http.Request) {
	var req eswagger.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user := eswagger.User{ID: 1, Username: req.Username, Email: req.Email}
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	user := eswagger.User{ID: 1, Username: "testuser", Email: "test@example.com"}
	json.NewEncoder(w).Encode(user)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	var req eswagger.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	user := eswagger.User{ID: 1, Username: req.Username, Email: req.Email}
	json.NewEncoder(w).Encode(user)
}

// Domain models

// ... (keep your existing handlers) ...

func main() {
	r := mux.NewRouter()

	// Initialize swagger generator with configuration
	swaggerGen := eswagger.NewGenerator(eswagger.Config{
		Title:       "User Management API",
		Description: "API for managing users",
		Version:     "1.0.0",
		BasePath:    "/api/v1",
		DocPath:     "doc",
	})

	// Register routes
	r.HandleFunc("/users", createUser).Methods("POST")
	r.HandleFunc("/users/{id}", getUser).Methods("GET")
	r.HandleFunc("/users/{id}", updateUser).Methods("PUT")
	// r.HandleFunc("/users/{id}", deleteUser).Methods("DELETE")

	// Generate swagger documentation from router
	if err := swaggerGen.GenerateFromRouter(r, eswagger.RouteMetadata{}); err != nil {
		log.Fatal("Failed to generate swagger documentation:", err)
	}

	// Serve swagger specification
	r.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swaggerGen.GetSwaggerSpec())
	})

	// Serve Swagger UI
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
		httpSwagger.DeepLinking(true),
	))

	if err := swaggerGen.SaveSwagger("yaml"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server starting on :8080")
	fmt.Println("Swagger UI available at: http://localhost:8080/swagger/")
	fmt.Printf("Swagger YAML available at: http://localhost:8080/swagger.%s\n", "yaml")
	fmt.Printf("Swagger JSON available at: http://localhost:8080/swagger.%s\n", "json")

	log.Fatal(http.ListenAndServe(":8080", r))
}
