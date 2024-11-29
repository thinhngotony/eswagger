package util

import (
	"encoding/json"

	"github.com/gookit/goutil"
)

func ToJSON(input interface{}) string {
	jsonRaw, err := json.Marshal(input)
	if err != nil {
		return goutil.String(input)
	}
	return string(jsonRaw)
}
