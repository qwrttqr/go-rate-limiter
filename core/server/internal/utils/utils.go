package utils

import (
	"fmt"
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
	"reflect"
)

func ExtractIdentificationKey(r *http.Request, rules []config.IdentificationStrategy) {
	for _, strategy := range rules {
		if strategy.Type == "authorization_header" {

		}
	}
}

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
