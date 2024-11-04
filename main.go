package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/go-openapi/spec"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Domain models
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// SwaggerGenerator handles the OpenAPI spec generation
type SwaggerGenerator struct {
	swagger *spec.Swagger
}

func NewSwaggerGenerator() *SwaggerGenerator {
	return &SwaggerGenerator{
		swagger: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Swagger: "2.0",
				Info: &spec.Info{
					InfoProps: spec.InfoProps{
						Title:       "API Documentation",
						Description: "Automatically generated API documentation",
						Version:     "1.0.0",
					},
				},
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
				Definitions: make(map[string]spec.Schema),
			},
		},
	}
}

// GenerateSchema creates OpenAPI schema from struct
func (sg *SwaggerGenerator) GenerateSchema(t reflect.Type) *spec.Schema {
	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
		},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}

		var fieldSchema spec.Schema
		switch field.Type.Kind() {
		case reflect.String:
			fieldSchema = *spec.StringProperty()
		case reflect.Int:
			fieldSchema = *spec.Int64Property()
			// Add more types as needed
		}

		schema.Properties[jsonTag] = fieldSchema
	}

	return schema
}

// AddEndpoint adds an endpoint to the swagger documentation
func (sg *SwaggerGenerator) AddEndpoint(path string, method string, summary string, reqType, respType reflect.Type) {
	operation := &spec.Operation{
		OperationProps: spec.OperationProps{
			Summary:  summary,
			Produces: []string{"application/json"},
		},
	}

	if reqType != nil {
		reqSchema := sg.GenerateSchema(reqType)
		sg.swagger.Definitions[reqType.Name()] = *reqSchema
		operation.Parameters = append(operation.Parameters, spec.Parameter{
			ParamProps: spec.ParamProps{
				Name: "body",
				In:   "body",
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Ref: spec.MustCreateRef("#/definitions/" + reqType.Name()),
					},
				},
			},
		})
	}

	if respType != nil {
		respSchema := sg.GenerateSchema(respType)
		sg.swagger.Definitions[respType.Name()] = *respSchema
		operation.Responses = &spec.Responses{
			ResponsesProps: spec.ResponsesProps{
				StatusCodeResponses: map[int]spec.Response{
					200: {
						ResponseProps: spec.ResponseProps{
							Schema: &spec.Schema{
								SchemaProps: spec.SchemaProps{
									Ref: spec.MustCreateRef("#/definitions/" + respType.Name()),
								},
							},
						},
					},
				},
			},
		}
	}

	pathItem := spec.PathItem{}
	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	}

	sg.swagger.Paths.Paths[path] = pathItem
}

// Example handlers
func createUser(w http.ResponseWriter, r *http.Request) {
	var user CreateUserRequest
	json.NewEncoder(w).Encode(User{ID: 1, Username: user.Username, Email: user.Email})
}

func getUser(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(User{ID: 1, Username: "testuser", Email: "test@example.com"})
}

func main() {
	r := mux.NewRouter()

	// Initialize swagger generator
	sg := NewSwaggerGenerator()

	// Add endpoints to swagger documentation
	sg.AddEndpoint("/users", "POST", "Create a new user",
		reflect.TypeOf(CreateUserRequest{}),
		reflect.TypeOf(User{}))

	sg.AddEndpoint("/users/{id}", "GET", "Get user by ID",
		nil,
		reflect.TypeOf(User{}))

	// Routes
	r.HandleFunc("/users", createUser).Methods("POST")
	r.HandleFunc("/users/{id}", getUser).Methods("GET")

	// Serve swagger documentation
	swaggerJSON, _ := json.MarshalIndent(sg.swagger, "", "  ")
	r.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(swaggerJSON)
	})

	// Serve Swagger UI
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
		httpSwagger.DeepLinking(true),
	))

	fmt.Println("Server starting on :8080")
	fmt.Println("Swagger UI available at: http://localhost:8080/swagger/")
	log.Fatal(http.ListenAndServe(":8080", r))
}
