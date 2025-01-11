package main

import (
	"bufio"
	"errors"
	"strings"
)

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
}

func GetUrlPath(request *Request) string {
	return strings.Split(request.Path, "?")[0]
}

func GetUrlParameters(request *Request) map[string]string {
	result := make(map[string]string)

	urlParts := strings.Split(request.Path, "?")
	if len(urlParts) != 2 {
		return result
	}

	for _, param := range strings.Split(urlParts[1], "&") {
		paramParts := strings.Split(param, "=")
		if len(paramParts) == 2 {
			result[paramParts[0]] = paramParts[1]
		}
	}

	return result
}

func GetCookieValueFromRequest(request *Request, name string) string {
	cookies, cookiesExists := request.Headers["Cookie"]
	if !cookiesExists {
		return ""
	}

	neededCookie := ""
	for _, cookie := range strings.Split(cookies, ";") {
		cookieParts := strings.Split(strings.TrimSpace(cookie), "=")
		if len(cookieParts) == 2 && cookieParts[0] == "drive" {
			neededCookie = cookieParts[1]
		}
	}

	return neededCookie
}

func ParseRequest(reader *bufio.ReadWriter) (Request, error) {
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return Request{}, err
	}

	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) < 3 {
		return Request{}, errors.New("less than 3 http params")
	}

	request := Request{Method: parts[0], Path: parts[1], Version: parts[2]}

	headers, err := parseHeaders(reader)
	if err != nil {
		return Request{}, nil
	}
	request.Headers = headers
	return request, nil
}

func parseHeaders(reader *bufio.ReadWriter) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return headers, nil
}
