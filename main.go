package main

import (
	"encoding/json"
	"log"
	"main/eswagger"
	"main/pkg/model"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

func CreateUser(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req []model.CreateUserStruct
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, err := s.CreateUser(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(data)
	}
}

func DeleteUser(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.DeleteUser(1)
		user := model.UserResponse{}
		json.NewEncoder(w).Encode(user)
	}
}

func UpdateUser(s model.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.UpdateUser(req)
		user := model.UserResponse{Info: []model.Info{{ID: 1, Username: req.Username, Email: req.Email}}}
		json.NewEncoder(w).Encode(user)
	}
}

func main() {
	r := mux.NewRouter()

	swaggerGen := eswagger.NewGenerator(eswagger.Config{
		Title:       "[REST] User Login API",
		Description: "This is a simple user login API",
		Version:     "1.0.0",
		BasePath:    "/api/v1",
		DocPath:     "doc",
	})

	var userSvc model.UserInterface
	// Register routes
	r.HandleFunc("/users", CreateUser(userSvc)).Methods("POST")
	r.HandleFunc("/users/{id}", DeleteUser(userSvc)).Methods("DELETE")
	r.HandleFunc("/users/{id}", UpdateUser(userSvc)).Methods("PUT")

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

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Swagger UI available at: http://localhost:8080/swagger/")

	log.Fatal(http.ListenAndServe(":8080", r))
}
