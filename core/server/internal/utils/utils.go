package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"qwrttqr-rate-limiter/core/server/internal/interfaces"
	"reflect"
)

func CheckRequiredFields(cfg config.AlgoSettings, requiredTags []string) error {
	value := reflect.ValueOf(cfg)
	type_ := reflect.TypeOf(cfg)

	for _, tag := range requiredTags {
		found := false
		for i := 0; i < value.NumField(); i++ {
			structField := type_.Field(i)
			yamlTag := structField.Tag.Get("yaml")
			if yamlTag == tag {
				found = true
				field := value.Field(i)
				if field.IsNil() {
					return fmt.Errorf("missing required configuration property: '%s'", tag)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown developer setting configuration tag: '%s'", tag)
		}
	}
	return nil
}

func ReadIncomingBody(r *http.Request) (*interfaces.IncomingBody, error) {
	defer r.Body.Close()

	var body interfaces.IncomingBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.ClientKey == "" {
		return nil, fmt.Errorf("client key should be not empty")
	}
	return &body, nil
}
