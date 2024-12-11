package eswagger

import (
	"strings"

	"github.com/go-openapi/spec"
)

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
