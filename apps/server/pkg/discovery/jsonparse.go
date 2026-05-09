package discovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

func extractJSONObject(raw string) ([]byte, error) {
	start := bytes.IndexByte([]byte(raw), '{')
	end := bytes.LastIndexByte([]byte(raw), '}')
	if start < 0 || end < start {
		return nil, errors.New("discovery: response did not contain a JSON object")
	}
	return []byte(raw[start : end+1]), nil
}

func decodeJSONObject(raw string, dst any) (map[string]json.RawMessage, error) {
	payload, err := extractJSONObject(raw)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return nil, fmt.Errorf("discovery: decode response fields: %w", err)
	}
	if err := json.Unmarshal(payload, dst); err != nil {
		return nil, fmt.Errorf("discovery: decode response body: %w", err)
	}
	return fields, nil
}

func requireKeys(fields map[string]json.RawMessage, keys ...string) error {
	for _, key := range keys {
		if _, ok := fields[key]; !ok {
			return fmt.Errorf("discovery: missing required key %q", key)
		}
	}
	return nil
}
