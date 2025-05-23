package main

import (
	json2 "encoding/json"
	"io"
)

func ParseRequestBody(body io.ReadCloser) map[string]string {
	var req map[string]string
	if err := json2.NewDecoder(body).Decode(&req); err != nil {
		return nil
	}
	return req
}
