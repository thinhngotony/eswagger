package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	Haha  string `json:"test_changed"`
	Email string `json:"email"`
}

type UpdateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// SwaggerGenerator handles the OpenAPI spec generation
type SwaggerGenerator struct {
	swagger *spec.Swagger
	docPath string
}

func NewSwaggerGenerator(docPath string) *SwaggerGenerator {
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
		docPath: docPath,
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
		case reflect.Bool:
			fieldSchema = *spec.BooleanProperty()
			// Add more types as needed
		}

		schema.Properties[jsonTag] = fieldSchema
	}

	return schema
}

// AddEndpoint adds an endpoint to the swagger documentation
func (sg *SwaggerGenerator) AddEndpoint(path string, method string, summary string, reqType, respType reflect.Type, requestExample, responseExample interface{}) {
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

		// if requestExample != nil {
		// 	reqExampleBytes, _ := json.MarshalIndent(requestExample, "", "  ")
		// 	operation.Parameters[0].ParamProps.AllowEmptyValue = string(reqExampleBytes)
		// }
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

		if responseExample != nil {
			respExampleBytes, _ := json.MarshalIndent(responseExample, "", "  ")

			// Create a new instance of spec.Response
			response := spec.Response{
				ResponseProps: spec.ResponseProps{
					Examples: map[string]interface{}{
						"application/json": string(respExampleBytes),
					},
				},
			}

			// Assign the new instance to operation.Responses.StatusCodeResponses[200]
			operation.Responses.StatusCodeResponses[200] = response
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
	case "PATCH":
		pathItem.Patch = operation
	}

	sg.swagger.Paths.Paths[path] = pathItem
}

// SaveSwagger saves the swagger specification to a file
func (sg *SwaggerGenerator) SaveSwagger(format string) {
	var data []byte
	var err error

	switch format {
	case "yaml":
		data, err = json.MarshalIndent(sg.swagger, "", "  ")
		if err != nil {
			log.Fatalf("Error marshalling Swagger spec: %v", err)
		}
		data, err = json.MarshalIndent(sg.swagger, "", "  ")
		if err != nil {
			log.Fatalf("Error marshalling Swagger spec: %v", err)
		}
	case "json":
		data, err = json.MarshalIndent(sg.swagger, "", "  ")
		if err != nil {
			log.Fatalf("Error marshalling Swagger spec: %v", err)
		}
	default:
		log.Fatalf("Invalid format specified: %s", format)
	}

	filePath := fmt.Sprintf("%s/swagger.%s", sg.docPath, format)
	os.WriteFile(filePath, data, 0644)
	fmt.Printf("Swagger spec saved to: %s\n", filePath)
}

// Example handlers
func createUser(w http.ResponseWriter, r *http.Request) {
	var user CreateUserRequest
	json.NewDecoder(r.Body).Decode(&user)
	json.NewEncoder(w).Encode(User{ID: 1, Username: user.Haha, Email: user.Email})
}

func getUser(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(User{})
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	var user UpdateUserRequest
	json.NewDecoder(r.Body).Decode(&user)
	json.NewEncoder(w).Encode(User{ID: 1, Username: user.Username, Email: user.Email})
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	r := mux.NewRouter()

	// Set document path
	docPath := "doc"

	// Initialize swagger generator
	sg := NewSwaggerGenerator(docPath)

	// Add endpoints to swagger documentation
	sg.AddEndpoint("/users", "POST", "Create a new user",
		reflect.TypeOf(CreateUserRequest{}),
		reflect.TypeOf(User{}),
		CreateUserRequest{"testuser", "test@example.com"},
		User{1, "testuser", "test@example.com", "2023-10-26T10:00:00Z"})

	sg.AddEndpoint("/users/{id}", "GET", "Get user by ID",
		nil,
		reflect.TypeOf(User{}),
		nil,
		User{1, "testuser", "test@example.com", "2023-10-26T10:00:00Z"})

	sg.AddEndpoint("/users/{id}", "PUT", "Update user by ID",
		reflect.TypeOf(UpdateUserRequest{}),
		reflect.TypeOf(User{}),
		UpdateUserRequest{"updateduser", "updated@example.com"},
		User{1, "updateduser", "updated@example.com", "2023-10-26T10:00:00Z"})

	sg.AddEndpoint("/users/{id}", "DELETE", "Delete user by ID",
		nil,
		nil,
		nil,
		nil)

	// Routes
	r.HandleFunc("/users", createUser).Methods("POST")
	r.HandleFunc("/users/{id}", getUser).Methods("GET")
	r.HandleFunc("/users/{id}", updateUser).Methods("PUT")
	r.HandleFunc("/users/{id}", deleteUser).Methods("DELETE")

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

	// Save swagger specs to files
	sg.SaveSwagger("yaml")
	sg.SaveSwagger("json")

	fmt.Println("Server starting on :8080")
	fmt.Println("Swagger UI available at: http://localhost:8080/swagger/")
	fmt.Printf("Swagger YAML available at: http://localhost:8080/swagger.%s\n", "yaml")
	fmt.Printf("Swagger JSON available at: http://localhost:8080/swagger.%s\n", "json")
	log.Fatal(http.ListenAndServe(":8080", r))
}
