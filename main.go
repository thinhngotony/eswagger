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

func CreateUser(s eswagger.UserSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req eswagger.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.CreateUser(req)
		user := eswagger.User{ID: 1, Username: req.Username, Email: req.Email}
		json.NewEncoder(w).Encode(user)
	}
}

func DeleteUser(s eswagger.UserSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req eswagger.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.DeleteUser(1)
		user := eswagger.User{ID: 1, Username: req.Username, Email: req.Email}
		json.NewEncoder(w).Encode(user)
	}
}

func updateUser(s eswagger.UserSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req eswagger.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.UpdateUser(req)
		user := eswagger.User{ID: 1, Username: req.Username, Email: req.Email}
		json.NewEncoder(w).Encode(user)
	}
}

func main() {
	r := mux.NewRouter()

	swaggerGen := eswagger.NewGenerator(eswagger.Config{
		Title:       "User Management API",
		Description: "API for managing users",
		Version:     "1.0.0",
		BasePath:    "/api/v1",
		DocPath:     "doc",
	})

	userSvc := eswagger.UserSvc{}
	// Register routes
	r.HandleFunc("/users", CreateUser(userSvc)).Methods("POST")
	r.HandleFunc("/users/{id}", DeleteUser(userSvc)).Methods("GET")
	r.HandleFunc("/users/{id}", updateUser(userSvc)).Methods("PUT")

	// Generate swagger documentation
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

	log.Fatal(http.ListenAndServe(":8080", r))
}
