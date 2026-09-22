package algos

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func ReadIncomingBody(r *http.Request) (*IncomingBody, error) {
	defer r.Body.Close()

	var body IncomingBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.ClientKey == "" {
		return nil, fmt.Errorf("client key should be not empty")
	}
	return &body, nil
}
