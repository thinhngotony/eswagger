package eswagger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
