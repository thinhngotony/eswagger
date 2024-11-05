package eswagger

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/go-openapi/spec"
	"github.com/gorilla/mux"
)

type Config struct {
	Title       string
	Description string
	Version     string
	BasePath    string
	DocPath     string
}

type EndpointMetadata struct {
	Summary     string
	Description string
	Tags        []string
	Examples    struct {
		Request  interface{}
		Response interface{}
	}
}

type RouteMetadata struct {
	Endpoints map[string]map[string]EndpointMetadata // path -> method -> metadata
}

func (g *Generator) convertMuxPathToSwagger(muxPath string) string {
	// Convert {param} to {param}
	return muxPath
}

func (g *Generator) addOperationToPathItem(pathItem *spec.PathItem, method string, operation *spec.Operation) {
	switch strings.ToUpper(method) {
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
}

func (g *Generator) SaveSwagger(format string) error {
	var data []byte
	var err error

	switch format {
	case "yaml", "json":
		data, err = json.MarshalIndent(g.swagger, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling Swagger spec: %v", err)
		}
	default:
		return fmt.Errorf("invalid format specified: %s", format)
	}

	filePath := fmt.Sprintf("%s/swagger.%s", g.config.DocPath, format)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing swagger file: %v", err)
	}

	log.Printf("Swagger spec saved to: %s\n", filePath)
	return nil
}

func (g *Generator) GetSwaggerSpec() *spec.Swagger {
	return g.swagger
}

func (g *Generator) generateSummary(handlerName, method string) string {
	parts := strings.Split(handlerName, ".")
	if len(parts) > 0 {
		funcName := parts[len(parts)-1]
		return strings.ToTitle(strings.Join(strings.Split(funcName, ""), " "))
	}
	return fmt.Sprintf("%s operation", method)
}

func (g *Generator) generateDescription(handlerName, method string) string {
	resource := g.extractResourceName(handlerName)
	switch method {
	case "GET":
		return fmt.Sprintf("Retrieve %s information", resource)
	case "POST":
		return fmt.Sprintf("Create a new %s", resource)
	case "PUT":
		return fmt.Sprintf("Update existing %s", resource)
	case "DELETE":
		return fmt.Sprintf("Delete %s", resource)
	default:
		return fmt.Sprintf("%s operation for %s", method, resource)
	}
}

func (g *Generator) extractResourceName(path string) string {

	log.Println(">>>>>>>>>>>>>>>>>:", path)
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part != "" && !strings.Contains(part, "{") {
			return strings.ToLower(part)
		}
	}
	return "resource"
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UpdateUserRequest struct {
	Username string `json:"update_username,omitempty"`
	Email    string `json:"update_email,omitempty"`
}

func (g *Generator) generateOperationFromHandler(handler interface{}, method string, path string) *spec.Operation {
	handlerValue := reflect.ValueOf(handler)
	handlerName := runtime.FuncForPC(handlerValue.Pointer()).Name()

	operation := &spec.Operation{
		OperationProps: spec.OperationProps{
			Summary:     g.generateSummary(handlerName, method),
			Description: g.generateDescription(handlerName, method),
			Tags:        []string{g.extractResourceName(path)},
			Produces:    []string{"application/json"},
			Consumes:    []string{"application/json"},
			Responses:   g.generateResponses(method, path),
		},
	}

	// Add request body for POST/PUT/PATCH
	if method == "POST" || method == "PUT" || method == "PATCH" {
		reqSchema := g.getRequestSchema(path, method)
		if reqSchema != "" {
			operation.Parameters = append(operation.Parameters, spec.Parameter{
				ParamProps: spec.ParamProps{
					Name:     "body",
					In:       "body",
					Required: true,
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef(reqSchema),
						},
					},
				},
			})
		}
	}

	// Add path parameters - Fixed version using proper Swagger 2.0 format
	if strings.Contains(path, "{id}") {
		operation.Parameters = append(operation.Parameters, spec.Parameter{
			SimpleSchema: spec.SimpleSchema{
				Type:   "integer",
				Format: "int64",
			},
			ParamProps: spec.ParamProps{
				Description: "ID of the resource",
				Name:        "id",
				In:          "path",
				Required:    true,
			},
		})
	}

	return operation
}

func (g *Generator) generateSchema(t reflect.Type) *spec.Schema {
	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
		},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		fieldSchema := g.getFieldSchema(field.Type)
		if fieldSchema != nil {
			schema.Properties[jsonTag] = *fieldSchema
			if !g.isOptionalField(field) {
				schema.Required = append(schema.Required, jsonTag)
			}
		}
	}

	return schema
}

func (g *Generator) isOptionalField(field reflect.StructField) bool {
	jsonTag := field.Tag.Get("json")
	return strings.Contains(jsonTag, "omitempty")
}

func (g *Generator) getFieldSchema(t reflect.Type) *spec.Schema {
	switch t.Kind() {
	case reflect.String:
		return &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"string"},
			},
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:   []string{"integer"},
				Format: "int64",
			},
		}
	case reflect.Float32, reflect.Float64:
		return &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:   []string{"number"},
				Format: "float",
			},
		}
	case reflect.Bool:
		return &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"boolean"},
			},
		}
	case reflect.Slice:
		items := g.getFieldSchema(t.Elem())
		if items != nil {
			return &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type:  []string{"array"},
					Items: &spec.SchemaOrArray{Schema: items},
				},
			}
		}
	default:
		//TODO("unhandled default case")
		return nil
	}
	return nil
}

type TypeMapping struct {
	RequestType  reflect.Type
	ResponseType reflect.Type
}

type Generator struct {
	swagger      *spec.Swagger
	config       Config
	routes       map[string]map[string]interface{}
	typeMappings map[string]map[string]TypeMapping // path -> method -> types
}

func NewGenerator(config Config) *Generator {
	return &Generator{
		swagger: &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Swagger: "2.0",
				Info: &spec.Info{
					InfoProps: spec.InfoProps{
						Title:       config.Title,
						Description: config.Description,
						Version:     config.Version,
					},
				},
				BasePath: config.BasePath,
				Paths: &spec.Paths{
					Paths: make(map[string]spec.PathItem),
				},
				Definitions: make(map[string]spec.Schema),
			},
		},
		config:       config,
		routes:       make(map[string]map[string]interface{}),
		typeMappings: make(map[string]map[string]TypeMapping),
	}
}

// RegisterEndpoint registers the request and response types for an endpoint
func (g *Generator) RegisterEndpoint(path, method string, requestType, responseType interface{}) {
	if g.typeMappings[path] == nil {
		g.typeMappings[path] = make(map[string]TypeMapping)
	}

	mapping := TypeMapping{}
	if requestType != nil {
		mapping.RequestType = reflect.TypeOf(requestType)
		// Pre-register the type in definitions
		g.registerType(requestType)
	}
	if responseType != nil {
		mapping.ResponseType = reflect.TypeOf(responseType)
		// Pre-register the type in definitions
		g.registerType(responseType)
	}

	g.typeMappings[path][strings.ToUpper(method)] = mapping
}

func (g *Generator) registerType(t interface{}) {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	schema := g.generateSchema(typ)
	g.swagger.Definitions[typ.Name()] = *schema
}

func (g *Generator) getRequestSchema(path, method string) string {
	if mapping, ok := g.typeMappings[path][method]; ok && mapping.RequestType != nil {
		return "#/definitions/" + mapping.RequestType.Name()
	}
	return ""
}

func (g *Generator) getResponseType(path, method string) reflect.Type {
	if mapping, ok := g.typeMappings[path][method]; ok {
		return mapping.ResponseType
	}
	return nil
}

func (g *Generator) generateResponses(method, path string) *spec.Responses {
	responses := &spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: make(map[int]spec.Response),
		},
	}

	var statusCode int
	var schema *spec.Schema

	switch method {
	case "GET":
		statusCode = http.StatusOK
	case "POST":
		statusCode = http.StatusCreated
	case "PUT":
		statusCode = http.StatusOK
	case "DELETE":
		statusCode = http.StatusNoContent
	default:
		statusCode = http.StatusOK
	}

	response := spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: http.StatusText(statusCode),
		},
	}

	// Get response type from registered mappings
	if respType := g.getResponseType(path, method); respType != nil && statusCode != http.StatusNoContent {
		schema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/" + respType.Name()),
			},
		}
		response.Schema = schema

		// Generate example response
		if example := g.generateExample(respType); example != nil {
			exampleBytes, err := json.Marshal(example)
			if err == nil {
				response.Examples = map[string]interface{}{
					"application/json": json.RawMessage(exampleBytes),
				}
			}
		}
	}

	responses.StatusCodeResponses[statusCode] = response
	return responses
}

// generateExample creates an example instance of the given type
func (g *Generator) generateExample(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Create a new instance of the type
	v := reflect.New(t).Interface()

	// You could add logic here to populate the instance with example data
	// based on field names or tags

	return v
}

// RegisterModels registers all model types that should appear in swagger definitions
func (g *Generator) RegisterModels(models ...interface{}) {
	for _, model := range models {
		typ := reflect.TypeOf(model)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		schema := g.generateSchema(typ)
		g.swagger.Definitions[typ.Name()] = *schema
	}
}

type MethodStructs struct {
	Input  interface{}
	Output interface{}
}

type UserSvc struct{}

func (m UserSvc) CreateUser(input CreateUserRequest) (User, error) {
	return User{ID: 1, Username: input.Username, Email: input.Email}, nil
}

func (m UserSvc) UpdateUser(input UpdateUserRequest) (User, error) {
	return User{ID: 1, Username: input.Username, Email: input.Email}, nil
}

func (m UserSvc) DeleteUser(id int) error {
	return nil
}

func GetInterfaceMethods(i interface{}) (map[string]MethodStructs, error) {
	methods := make(map[string]MethodStructs)
	val := reflect.ValueOf(i)
	typ := reflect.TypeOf(i)

	for j := 0; j < val.NumMethod(); j++ {
		method := val.Method(j)
		methodType := method.Type()
		methodName := typ.Method(j).Name

		// Check for methods with an input and output
		if methodType.NumIn() == 1 && methodType.NumOut() >= 1 {
			inputType := methodType.In(0)
			outputType := methodType.Out(0)

			// Instantiate the input and output structs if they are structs
			var inputInstance, outputInstance interface{}
			if inputType.Kind() == reflect.Struct {
				inputInstance = reflect.New(inputType).Elem().Interface()
			}
			if outputType.Kind() == reflect.Struct {
				outputInstance = reflect.New(outputType).Elem().Interface()
			}

			// Store the instances in the map
			methods[methodName] = MethodStructs{
				Input:  inputInstance,
				Output: outputInstance,
			}
		}
	}
	return methods, nil
}

func (g *Generator) GenerateFromRouter(router *mux.Router, _ RouteMetadata) error {

	return router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil {
			log.Panic(err)
			return nil
		}

		rest := UserSvc{}
		data, err := GetInterfaceMethods(rest)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}

		for methodName, structs := range data {
			fmt.Printf("→ Method: %s\n", methodName)
			fmt.Printf(" Input Struct: %+v\n", structs.Input)
			fmt.Printf(" Output Struct: %+v\n", structs.Output)

			// Extract and store handler function names
			handler := route.GetHandler()
			handlerName := g.getHandlerFunctionName(handler)

			fmt.Printf("→ Service name: %+v\n", handlerName)

			if strings.Contains(handlerName, methodName) {
				g.RegisterEndpoint(pathTemplate, strings.Join(methods, ""), structs.Input, structs.Output)

			}
		}

		if g.routes[pathTemplate] == nil {
			g.routes[pathTemplate] = make(map[string]interface{})
		}

		handler := route.GetHandler()

		pathItem := spec.PathItem{}

		for _, method := range methods {
			g.routes[pathTemplate][method] = handler
			operation := g.generateOperationFromHandler(handler, method, pathTemplate)
			g.addOperationToPathItem(&pathItem, method, operation)
		}

		swaggerPath := g.convertMuxPathToSwagger(pathTemplate)
		g.swagger.Paths.Paths[swaggerPath] = pathItem

		return nil
	})
}

// Helper to extract only the function name from the handler
func (g *Generator) getHandlerFunctionName(handler http.Handler) string {
	if handlerFunc, ok := handler.(http.HandlerFunc); ok {
		fullName := runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name()
		return fullName
		// parts := strings.Split(fullName, ".")
		// // Get last part of the name, then split on "func" in case of closures
		// lastPart := parts[len(parts)-1]
		// if idx := strings.Index(lastPart, ".func"); idx != -1 {
		// 	lastPart = lastPart[:idx] // Remove the ".funcX" suffix
		// }
		// return lastPart // Return only the function name
	}
	return "unknownHandler"
}
