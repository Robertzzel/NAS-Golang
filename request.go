package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
}

type Cookie struct {
	username string
	value    string
	expires  time.Time
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
			p0 := strings.ReplaceAll(strings.TrimSpace(paramParts[0]), "%20", " ")
			p1 := strings.ReplaceAll(strings.TrimSpace(paramParts[1]), "%20", " ")
			result[p0] = p1
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
		if len(cookieParts) == 2 && cookieParts[0] == name {
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

func GetCookie(activeCookies []Cookie, request *Request) (*Cookie, error) {
	neededCookieValue := GetCookieValueFromRequest(request, "drive")
	if neededCookieValue == "" {
		return nil, errors.New("cookie not found")
	}

	for _, cookie := range activeCookies {
		if cookie.value == neededCookieValue && cookie.expires.After(time.Now()) {
			return &cookie, nil
		}
	}

	return nil, errors.New("cookie not found")
}

func ParseFormBody(body string) map[string]string {
	result := make(map[string]string)

	for _, line := range strings.Split(body, "&") {
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result
}

func handleMultipartBody(body *bufio.Reader, contentType string, base string) error {
	const boundaryPrefix = "boundary="
	boundaryIndex := strings.Index(contentType, boundaryPrefix)
	if boundaryIndex == -1 {
		return fmt.Errorf("no boundary found in Content-Type")
	}
	boundary := contentType[boundaryIndex+len(boundaryPrefix):]

	mr := multipart.NewReader(body, boundary)

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading part: %v", err)
		}

		if part.FileName() != "" {

			fileName := filepath.Join(base, part.FileName())
			dst, err := os.Create(fileName)
			if err != nil {
				return fmt.Errorf("error creating file: %v", err)
			}
			defer dst.Close()

			_, err = io.Copy(dst, part)
			if err != nil {
				return fmt.Errorf("error saving file: %v", err)
			}
		} else {
			buf := new(bytes.Buffer)
			buf.ReadFrom(part)
		}
	}

	return nil
}
