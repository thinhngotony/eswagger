package eswagger

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"unicode"

	"github.com/gorilla/mux"

	"main/pkg/model"

	"github.com/fatih/structtag"
	"github.com/go-openapi/spec"
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

func (g *Generator) addOperationToPathItem(pathItem *spec.PathItem, method string, operation *spec.Operation) {
	switch strings.ToUpper(method) {
	case "GET":
		if pathItem.Get == nil {
			pathItem.Get = operation
		}
	case "POST":
		if pathItem.Post == nil {
			pathItem.Post = operation
		}
	case "PUT":
		if pathItem.Put == nil {
			pathItem.Put = operation
		}
	case "DELETE":
		if pathItem.Delete == nil {
			pathItem.Delete = operation
		}
	case "PATCH":
		if pathItem.Patch == nil {
			pathItem.Patch = operation
		}
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

func (g *Generator) extractResourceName(path string) string {

	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part != "" && !strings.Contains(part, "{") {
			return strings.ToLower(part)
		}
	}
	return "resource"
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
	//if method == "POST" || method == "PUT" || method == "PATCH" {
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
	//}

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

func (g *Generator) generateRequestOld(t reflect.Type) *spec.Schema {

	// Safely handle nil and pointer types
	if t == nil {
		return nil
	}

	// Handle pointer types by unwrapping them
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

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

		// Handle pointer fields
		fieldType := field.Type
		isPointer := false
		if fieldType.Kind() == reflect.Ptr {
			isPointer = true
			fieldType = fieldType.Elem()
		}

		fieldSchema := g.getFieldSchema(fieldType)
		if fieldSchema != nil {
			// If the field is a pointer, mark it as nullable
			if isPointer {
				fieldSchema.Nullable = true
			}

			fieldSchema.Description = field.Tag.Get("doc") // Retrieve the "doc" tag value and set it as the Description
			fieldSchema.Example = field.Tag.Get("example") // Retrieve the "example" tag value and set it as the Example

			schema.Properties[jsonTag] = *fieldSchema

			// Only add to required if it's not a pointer field
			if !isPointer && g.isRequiredField(field) {
				schema.Required = append(schema.Required, jsonTag)
			}
		}
	}

	return schema
}

func (g *Generator) generateRequest(t reflect.Type) *spec.Schema {
	// Safely handle nil and pointer types
	if t == nil {
		return nil
	}

	// Handle pointer types by unwrapping them
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type:       []string{"object"},
			Properties: make(map[string]spec.Schema),
		},
	}

	// Use a map to track processed field names to handle embedded structs
	processedFields := make(map[string]bool)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Handle embedded structs
		if field.Anonymous {
			embeddedSchema := g.generateRequest(field.Type)
			if embeddedSchema != nil {
				for propName, propSchema := range embeddedSchema.Properties {
					if _, exists := processedFields[propName]; !exists {
						schema.Properties[propName] = propSchema
						processedFields[propName] = true
					}
				}
				// Add any required fields from embedded struct
				schema.Required = append(schema.Required, embeddedSchema.Required...)
			}
			continue
		}

		jsonTag, _ := field.Tag.Lookup("json")
		jsonParts := strings.Split(jsonTag, ",")
		fieldName := jsonParts[0]

		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Handle pointer fields
		fieldType := field.Type
		isPointer := false
		if fieldType.Kind() == reflect.Ptr {
			isPointer = true
			fieldType = fieldType.Elem()
		}

		fieldSchema := g.getFieldSchema(fieldType)
		if fieldSchema != nil {
			// If the field is a pointer, mark it as nullable
			if isPointer {
				fieldSchema.Nullable = true
			}

			// Set description from doc tag
			docTag := field.Tag.Get("doc")
			if docTag != "" {
				fieldSchema.Description = docTag
			}

			// Set example from example tag
			exampleTag := field.Tag.Get("example")
			if exampleTag != "" {
				fieldSchema.Example = exampleTag
			}

			schema.Properties[fieldName] = *fieldSchema

			// Only add to required if it's not a pointer field
			if !isPointer && g.isRequiredField(field) {
				schema.Required = append(schema.Required, fieldName)
			}
		}
	}

	return schema
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
		//if example := g.generateExample(respType); example != nil {
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

func (g *Generator) isRequiredField(field reflect.StructField) bool {
	jsonTag := field.Tag.Get("validate")
	return strings.Contains(jsonTag, "required")
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
	swagger          *spec.Swagger
	config           Config
	routes           map[string]map[string]interface{}
	typeMappings     map[string]map[string]TypeMapping // path -> method -> types
	exampleGenerator *ExampleGenerator
}

// DocTag represents the structure for documentation tags
type DocTag struct {
	Description string   `json:"description"`
	Example     string   `json:"example"`
	Required    bool     `json:"required"`
	Format      string   `json:"format"`
	Enum        []string `json:"enum,omitempty"`
}

// FieldMetadata stores field documentation
type FieldMetadata struct {
	Description string
	Example     interface{}
	Required    bool
	Format      string
	Enum        []string
}

// ExampleGenerator handles example generation for different types
type ExampleGenerator struct {
	customExamples map[reflect.Type]interface{}
}

func NewExampleGenerator() *ExampleGenerator {
	return &ExampleGenerator{
		customExamples: make(map[reflect.Type]interface{}),
	}
}

// RegisterCustomExample allows registering custom examples for specific types
func (g *ExampleGenerator) RegisterCustomExample(t reflect.Type, example interface{}) {
	g.customExamples[t] = example
}

// GenerateExample creates an example value for a given type
func (g *ExampleGenerator) GenerateExample(t reflect.Type) interface{} {
	// Check for custom examples first
	if example, exists := g.customExamples[t]; exists {
		return example
	}

	switch t.Kind() {
	case reflect.String:
		return "example_string"
	case reflect.Int, reflect.Int64:
		return 42
	case reflect.Float64:
		return 42.42
	case reflect.Bool:
		return true
	case reflect.Struct:
		return g.generateStructExample(t)
	case reflect.Slice:
		return g.generateSliceExample(t)
	case reflect.Map:
		return g.generateMapExample(t)
	case reflect.Ptr:
		return g.GenerateExample(t.Elem())
	default:
		return nil
	}
}

func (g *ExampleGenerator) generateStructExample(t reflect.Type) interface{} {
	v := reflect.New(t).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if example := g.getExampleFromTag(field); example != nil {
			v.Field(i).Set(reflect.ValueOf(example))
		} else {
			fieldExample := g.GenerateExample(field.Type)
			if fieldExample != nil {
				v.Field(i).Set(reflect.ValueOf(fieldExample))
			}
		}
	}
	return v.Interface()
}

func (g *ExampleGenerator) generateSliceExample(t reflect.Type) interface{} {
	elemExample := g.GenerateExample(t.Elem())
	if elemExample == nil {
		return nil
	}

	slice := reflect.MakeSlice(t, 1, 1)
	slice.Index(0).Set(reflect.ValueOf(elemExample))
	return slice.Interface()
}

func (g *ExampleGenerator) generateMapExample(t reflect.Type) interface{} {
	m := reflect.MakeMap(t)
	keyExample := g.GenerateExample(t.Key())
	valueExample := g.GenerateExample(t.Elem())

	if keyExample != nil && valueExample != nil {
		m.SetMapIndex(reflect.ValueOf(keyExample), reflect.ValueOf(valueExample))
	}

	return m.Interface()
}

func (g *ExampleGenerator) getExampleFromTag(field reflect.StructField) interface{} {
	docTag := field.Tag.Get("doc")
	if docTag == "" {
		return nil
	}

	tags, err := structtag.Parse(docTag)
	if err != nil {
		return nil
	}

	example, err := tags.Get("example")
	if err != nil {
		return nil
	}
	log.Println(example)
	return nil
	// return convertExample(example.Value, field.Type)
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
		config:           config,
		routes:           make(map[string]map[string]interface{}),
		typeMappings:     make(map[string]map[string]TypeMapping),
		exampleGenerator: NewExampleGenerator(),
	}
}

// RegisterEndpoint registers the request and response types for an endpoint
func (g *Generator) RegisterEndpoint(path, method string, requestType, responseType interface{}) {
	if g.typeMappings[path] == nil {
		g.typeMappings[path] = make(map[string]TypeMapping)
	}

	mapping := TypeMapping{}
	if requestType != nil {
		// Handle both pointer and non-pointer types
		reqType := reflect.TypeOf(requestType)

		// If it's a pointer, get the underlying type
		if reqType.Kind() == reflect.Ptr {
			reqType = reqType.Elem()
		}

		log.Println(">>> Request type:", reqType)
		mapping.RequestType = reqType
		g.registerType(requestType)
	}

	if responseType != nil {
		// Handle both pointer and non-pointer types
		respType := reflect.TypeOf(responseType)

		// If it's a pointer, get the underlying type
		if respType.Kind() == reflect.Ptr {
			respType = respType.Elem()
		}

		mapping.ResponseType = respType
		g.registerType(responseType)
	}

	g.typeMappings[path][strings.ToUpper(method)] = mapping
}

func (g *Generator) registerType(t interface{}) {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	schema := g.generateRequest(typ)
	g.swagger.Definitions[typ.Name()] = *schema
}

func (g *Generator) getRequestSchema(path, method string) string {
	log.Printf("[INFO] getRequestSchema called with path: %s, method: %s", path, method)
	if mapping, ok := g.typeMappings[path][method]; ok {
		log.Printf("[INFO] Mapping found for path: %s, method: %s", path, method)
		if mapping.RequestType != nil {
			log.Printf("[INFO] Request type found for path: %s, method: %s - %s", path, method, mapping.RequestType.Name())
			return "#/definitions/" + mapping.RequestType.Name()
		} else {
			log.Printf("[ERROR] No request type found for path: %s, method: %s", path, method)
		}
	} else {
		log.Printf("[ERROR] No mapping found for path: %s, method: %s", path, method)
	}
	return ""
}

func (g *Generator) getResponseType(path, method string) reflect.Type {
	if mapping, ok := g.typeMappings[path][method]; ok {
		return mapping.ResponseType
	}
	return nil
}

type MethodStructs struct {
	Input  interface{}
	Output interface{}
}

type UserSvc struct{}

func (m UserSvc) CreateUser(input model.CreateUserRequest) (model.UserResponse, error) {
	return model.UserResponse{Info: []model.Info{{ID: 1, Username: input.Username, Email: input.Email}}}, nil
}

func (m UserSvc) UpdateUser(input model.UpdateUserRequest) (model.UserResponse, error) {
	return model.UserResponse{Info: []model.Info{{ID: 1, Username: input.Username, Email: input.Email}}}, nil
}

func (m UserSvc) DeleteUser(_ int) error {
	return nil
}

func GetInterfaceTypeMethods(interfaceType reflect.Type) (map[string]*MethodStructs, error) {
	methods := make(map[string]*MethodStructs)

	// Check if input is nil
	if interfaceType == nil {
		return nil, fmt.Errorf("input type is nil")
	}

	// Ensure we're working with an interface type
	if interfaceType.Kind() != reflect.Interface {
		return nil, fmt.Errorf("input type is not an interface (got %v)", interfaceType.Kind())
	}

	// Iterate over all methods in the interface
	for i := 0; i < interfaceType.NumMethod(); i++ {
		method := interfaceType.Method(i)
		methodType := method.Type

		// Initialize method struct
		methodStruct := &MethodStructs{}

		// Find first struct-like input type (skipping receiver)
		var inputInstance interface{}
		for j := 0; j < methodType.NumIn(); j++ {
			inputType := methodType.In(j)
			if inputType.Kind() == reflect.Ptr {
				inputInstance = reflect.New(inputType.Elem()).Interface()
				break
			} else if inputType.Kind() == reflect.Struct {
				inputInstance = reflect.New(inputType).Elem().Interface()
				break
			}
		}
		methodStruct.Input = inputInstance

		// Find first struct-like output type
		var outputInstance interface{}
		for j := 0; j < methodType.NumOut(); j++ {
			outputType := methodType.Out(j)
			if outputType.Kind() == reflect.Ptr {
				outputInstance = reflect.New(outputType.Elem()).Elem().Interface()
				break
			} else if outputType.Kind() == reflect.Struct {
				outputInstance = reflect.New(outputType).Elem().Interface()
				break
			}
		}
		methodStruct.Output = outputInstance

		// Add method if either input or output is found
		if methodStruct.Input != nil || methodStruct.Output != nil {
			methods[method.Name] = methodStruct
		}
	}

	return methods, nil
}

// GetInterfaceMethodsFromType Helper function to get interface type from an interface definition
func GetInterfaceMethodsFromType(i interface{}) (map[string]*MethodStructs, error) {
	t := reflect.TypeOf(i)
	if t == nil {
		return nil, fmt.Errorf("input is nil")
	}

	if t.Kind() != reflect.Ptr || (t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Interface) {
		return nil, fmt.Errorf("input is not an interface or pointer to interface")
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return GetInterfaceTypeMethods(t)
}

func (g *Generator) GenerateFromRouter(router *mux.Router, _ RouteMetadata) error {
	pathItems := make(map[string]spec.PathItem)

	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil {
			log.Printf("Warning: couldn't get methods for route %s: %v", pathTemplate, err)
			return nil
		}

		// Get existing PathItem or create new one
		pathItem, exists := pathItems[pathTemplate]
		if !exists {
			pathItem = spec.PathItem{}
		}

		methodStructs, err := GetInterfaceMethodsFromType((*model.UserInterface)(nil))
		if err != nil {
			log.Printf("Warning: couldn't get interface methods: %v", err)
			return nil
		}

		handler := route.GetHandler()
		handlerName := g.getHandlerFunctionName(handler)

		// Match handler with method structs and register endpoints
		for methodName, structs := range methodStructs {
			if strings.Contains(handlerName, methodName) {
				for _, method := range methods {
					log.Printf("Registering endpoint [%v] for method [%v], input [%v], output [%v]", pathTemplate, method, structs.Input, structs.Output)
					g.RegisterEndpoint(pathTemplate, method, structs.Input, structs.Output)
				}
			}
		}

		// Generate operations for each HTTP method
		for _, method := range methods {
			operation := g.generateOperationFromHandler(handler, method, pathTemplate)
			g.addOperationToPathItem(&pathItem, method, operation)
		}

		// Store updated PathItem
		pathItems[pathTemplate] = pathItem
		g.swagger.Paths.Paths[pathTemplate] = pathItem

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking routes: %v", err)
	}

	return nil
}

func (g *Generator) cleanHandlerName(handlerName string) string {
	// Split the handler name by "."

	parts := strings.Split(handlerName, ".")
	if len(parts) == 0 {
		return ""
	}

	// Get the second to last part which contains the function name
	var name string
	if len(parts) > 1 {
		name = parts[len(parts)-2] // This will now capture "DeleteUser" from "main.main.DeleteUser.func3"
	} else {
		name = parts[len(parts)-1]
	}

	// Remove ".func1", ".func2", etc. suffixes from closures if they are in the last part
	if lastPart := parts[len(parts)-1]; strings.HasPrefix(lastPart, "func") {
		// If the last part starts with "func", we ignore it
		// name will remain as is since we took it from the second to last part
	}

	// Convert to title case and split camelCase
	return g.splitCamelCase(name)
}

// Add this helper to split camelCase properly
func (g *Generator) splitCamelCase(s string) string {
	// Handle common prefixes
	s = strings.TrimPrefix(s, "get")
	s = strings.TrimPrefix(s, "post")
	s = strings.TrimPrefix(s, "put")
	s = strings.TrimPrefix(s, "delete")
	s = strings.TrimPrefix(s, "patch")

	var words []string
	word := ""

	for i, r := range s {
		if i > 0 && (unicode.IsUpper(r) || unicode.IsNumber(r)) {
			if len(word) > 0 {
				words = append(words, word)
			}
			word = string(r)
		} else {
			word += string(r)
		}
	}

	if len(word) > 0 {
		words = append(words, word)
	}

	return strings.Join(words, " ")
}

// Replace the existing generateSummary method
func (g *Generator) generateSummary(handlerName, method string) string {
	cleanName := g.cleanHandlerName(handlerName)

	if cleanName == "" {
		return fmt.Sprintf("%s operation", method)
	}

	return cleanName

}

// Update the getHandlerFunctionName method
func (g *Generator) getHandlerFunctionName(handler http.Handler) string {
	if handlerFunc, ok := handler.(http.HandlerFunc); ok {
		fullName := runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name()
		return strings.Replace(g.cleanHandlerName(fullName), " ", "", -1)
	}
	return "UnknownHandler"
}

// Modify the generateDescription method to use cleanHandlerName
func (g *Generator) generateDescription(handlerName, method string) string {
	// Get clean name without the func2 suffix and properly formatted
	cleanName := g.cleanHandlerName(handlerName)

	// Extract the resource name - typically would be "User" from "CreateUser"
	resource := g.getResourceFromHandler(cleanName)

	return resource

}

func ExtractFuncName(input string) string {
	pattern := `(?:\w+\.)*(\w+)\.func\d+`

	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)

	if len(matches) > 1 {
		// Extract the function name
		functionName := matches[1]
		log.Println("Extracted function name:", functionName)
		return functionName
	}
	log.Println("No match found.")
	return input

}

// Add this new helper method to extract the resource name from the handler name
func (g *Generator) getResourceFromHandler(handlerName string) string {
	// Remove common prefixes

	name := strings.TrimPrefix(strings.ToLower(handlerName), "create")

	name = strings.TrimPrefix(name, "update")
	name = strings.TrimPrefix(name, "delete")
	name = strings.TrimPrefix(name, "get")
	name = strings.TrimPrefix(name, "post")
	name = strings.TrimPrefix(name, "put")

	// Clean up any remaining spaces and convert first character to lower case
	name = strings.TrimSpace(handlerName)
	if len(name) > 0 {
		return strings.ToLower(name[:1]) + name[1:]
	}
	return "resource"
}
