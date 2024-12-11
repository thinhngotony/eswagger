package main

import (
	"encoding/json"
	"log"
	"main/eswagger"
	"main/pkg/model"
	"main/pkg/service"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

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
	r.HandleFunc("/users", service.CreateUser(userSvc)).Methods("POST")
	r.HandleFunc("/users/{id}", service.DeleteUser(userSvc)).Methods("DELETE")
	r.HandleFunc("/users/{id}", service.UpdateUser(userSvc)).Methods("PUT")
	r.HandleFunc("/users/CreateUserPointerSliceToPointerResponse", service.CreateUserPointerSliceToPointerResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/NotWork_CreateUserSliceToPointerResponse", service.NotWork_CreateUserSliceToPointerResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserStructToPointerResponse", service.CreateUserStructToPointerResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserPointerSliceToSliceResponse", service.CreateUserPointerSliceToSliceResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserStructToSliceResponse", service.CreateUserStructToSliceResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserPointerSliceToNonPointerResponse", service.CreateUserPointerSliceToNonPointerResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserStructToNonPointerResponse", service.CreateUserStructToNonPointerResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserPointerSliceToNonPointerSliceResponse", service.CreateUserPointerSliceToNonPointerSliceResponse(userSvc)).Methods(http.MethodPost)
	r.HandleFunc("/users/CreateUserStructToNonPointerSliceResponse", service.CreateUserStructToNonPointerResponse(userSvc)).Methods(http.MethodPost)

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
