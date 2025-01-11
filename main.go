package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HOST             = "localhost"
	PORT             = "8080"
	UPLOAD_DIR       = "./uploads"
	USERS_FILE       = "./users.csv"
	CERTIFICATE_FILE = ""
	KEY_FILE         = ""
)

func main() {
	users, err := ParseUsersFile(USERS_FILE)
	if err != nil {
		return
	}

	for _, user := range users {
		path := filepath.Join(UPLOAD_DIR, user[0])
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
				fmt.Printf("Error creating upload directory: %v\n", err)
				return
			}
		}
	}

	cert, err := tls.LoadX509KeyPair(CERTIFICATE_FILE, KEY_FILE)
	if err != nil {
		log.Fatal("Error loading certificate. ", err)
	}

	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := tls.Listen("tcp", HOST+":"+PORT, tlsCfg)
	if err != nil {
		return
	}
	defer listener.Close()

	fmt.Printf("Server listening on %s:%s\n", HOST, PORT)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		readWriter := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		request, err := ParseRequest(readWriter)
		if err != nil {
			_ = conn.Close()
			continue
		}

		handleRequest(&request, readWriter)

		_ = readWriter.Flush()
		_ = conn.Close()
	}
}

type Cookie struct {
	username string
	value    string
	expires  time.Time
}

var existingCookies = make([]Cookie, 0)

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

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	sb := strings.Builder{}
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

func ParseUsersFile(path string) ([][]string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(file)
	result := make([][]string, 0)

	for _, line := range strings.Split(text, "\n") {
		parts := strings.Split(line, ",")
		if len(parts) == 2 {
			result = append(result, parts)
		}
	}

	return result, nil
}

func DisplayRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := GetCookie(existingCookies, request)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
		return
	}

	urlParameters := GetUrlParameters(request)

	parameterPath, pathExists := urlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	info, err := os.Stat(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}

	if !info.IsDir() {
		_ = sendResponse(conn, "400 Bad Request", []byte("path not found"))
		return
	}

	SendDirectoryStructure(conn, path, parameterPath)
}

func DownloadRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := GetCookie(existingCookies, request)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
		return
	}

	urlParameters := GetUrlParameters(request)

	parameterPath, pathExists := urlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	info, err := os.Stat(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}

	if info.IsDir() {
		SendDirectoryAsZip(path, conn)
	} else {
		SendFile(conn, path)
	}
}

func handleRequest(request *Request, conn *bufio.ReadWriter) {
	urlPath := GetUrlPath(request)

	if strings.Contains(urlPath, "..") {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}

	if urlPath == "/log" {
		LoginRoute(request, conn)
		return
	}

	_, err := GetCookie(existingCookies, request)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
		return
	}

	if urlPath == "/download" {
		DownloadRoute(request, conn)
		return
	}

	if urlPath == "/display" {
		DisplayRoute(request, conn)
		return
	}
}
