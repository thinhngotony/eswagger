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

type Generator struct {
	swagger *spec.Swagger
	config  Config
	routes  map[string]map[string]interface{} // path -> method -> handler
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

func (g *Generator) extractPathParameters(method string) []spec.Parameter {
	var params []spec.Parameter
	// Add common parameters like ID for path parameters
	if strings.Contains(method, "{id}") {
		params = append(params, spec.Parameter{
			ParamProps: spec.ParamProps{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   spec.Int64Property(),
			},
		})
	}
	return params
}

func (g *Generator) addRequestBody(operation *spec.Operation, reqType reflect.Type, example interface{}) {
	schema := g.generateSchema(reqType)
	g.swagger.Definitions[reqType.Name()] = *schema

	operation.Parameters = append(operation.Parameters, spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:     "body",
			In:       "body",
			Required: true,
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef("#/definitions/" + reqType.Name()),
				},
			},
		},
	})

	if example != nil {
		exampleBytes, _ := json.MarshalIndent(example, "", "  ")
		operation.Parameters[len(operation.Parameters)-1].Example = json.RawMessage(exampleBytes)
	}
}

func (g *Generator) addResponse(operation *spec.Operation, respType reflect.Type, example interface{}) {
	schema := g.generateSchema(respType)
	g.swagger.Definitions[respType.Name()] = *schema

	response := spec.Response{
		ResponseProps: spec.ResponseProps{
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef("#/definitions/" + respType.Name()),
				},
			},
		},
	}

	if example != nil {
		exampleBytes, _ := json.MarshalIndent(example, "", "  ")
		response.Examples = map[string]interface{}{
			"application/json": json.RawMessage(exampleBytes),
		}
	}

	operation.Responses = &spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: map[int]spec.Response{
				http.StatusOK: response,
			},
		},
	}
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
		config: config,
		routes: make(map[string]map[string]interface{}),
	}
}

func (g *Generator) extractRequestType(handlerType reflect.Type) reflect.Type {
	// Look for *http.Request parameter
	for i := 0; i < handlerType.NumIn(); i++ {
		paramType := handlerType.In(i)
		if paramType.Kind() == reflect.Ptr && paramType.Elem().Name() == "Request" {
			return paramType
		}
	}
	return nil
}

func (g *Generator) addRequestBodyWithoutExample(operation *spec.Operation, reqType reflect.Type) {
	schema := g.generateSchema(reqType)
	g.swagger.Definitions[reqType.Name()] = *schema

	operation.Parameters = append(operation.Parameters, spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:     "body",
			In:       "body",
			Required: true,
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.MustCreateRef("#/definitions/" + reqType.Name()),
				},
			},
		},
	})
}

func (g *Generator) addDefaultResponse(operation *spec.Operation, method string) {
	var statusCode int
	switch method {
	case "POST":
		statusCode = http.StatusCreated
	case "DELETE":
		statusCode = http.StatusNoContent
	default:
		statusCode = http.StatusOK
	}

	operation.Responses = &spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: map[int]spec.Response{
				statusCode: {
					ResponseProps: spec.ResponseProps{
						Description: http.StatusText(statusCode),
					},
				},
			},
		},
	}
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
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

func (g *Generator) GenerateFromRouter(router *mux.Router, _ RouteMetadata) error {
	// Pre-register all struct types used in handlers
	g.registerTypes(
		User{},
		CreateUserRequest{},
		UpdateUserRequest{},
	)

	return router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil {
			return nil
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

func (g *Generator) registerTypes(types ...interface{}) {
	for _, t := range types {
		typ := reflect.TypeOf(t)
		schema := g.generateSchema(typ)
		g.swagger.Definitions[typ.Name()] = *schema
	}
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

func (g *Generator) generateOperationFromHandler1(handler interface{}, method string, path string) *spec.Operation {
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

	// Add path parameters - Fixed version
	if strings.Contains(path, "{id}") {
		operation.Parameters = append(operation.Parameters, spec.Parameter{
			ParamProps: spec.ParamProps{
				Description: "ID of the resource",
				Name:        "id",
				In:          "path",
				Required:    true,
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Type:   []string{"integer"},
						Format: "int64",
					},
				},
			},
		})
	}

	return operation
}

func (g *Generator) getRequestSchema(path, method string) string {
	switch {
	case path == "/users" && method == "POST":
		return "#/definitions/CreateUserRequest"
	case strings.HasPrefix(path, "/users/") && method == "PUT":
		return "#/definitions/UpdateUserRequest"
	default:
		return ""
	}
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
		schema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/User"),
			},
		}
	case "POST":
		statusCode = http.StatusCreated
		schema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/User"),
			},
		}
	case "PUT":
		statusCode = http.StatusOK
		schema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/User"),
			},
		}
	case "DELETE":
		statusCode = http.StatusNoContent
		schema = nil
	default:
		statusCode = http.StatusOK
	}

	response := spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: http.StatusText(statusCode),
		},
	}

	if schema != nil {
		response.Schema = schema

		//// Add example response if available
		//if example := getExampleResponse(method, path); example != nil {
		//	exampleBytes, err := json.Marshal(example)
		//	if err == nil {
		//		response.Examples = map[string]interface{}{
		//			"application/json": json.RawMessage(exampleBytes),
		//		}
		//	}
		//}
	}

	responses.StatusCodeResponses[statusCode] = response
	return responses
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
