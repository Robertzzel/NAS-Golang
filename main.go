package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	HOST             = "localhost"
	PORT             = "8080"
	UPLOAD_DIR       = "./uploads"
	USERS_FILE       = "./users.csv"
	CERTIFICATE_FILE = "./cert.pem"
	KEY_FILE         = "./key.pem"
)

var existingCookies = make([]Cookie, 0)

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

		fmt.Printf("Accepted connection from %s\n", conn.RemoteAddr())
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

	switch urlPath {
	case "/download":
		DownloadRoute(request, conn)
	case "/upload":
		UploadRoute(request, conn)
	case "/display":
		DisplayRoute(request, conn)
	case "/delete":
		DeleteRoute(request, conn)
	case "/create-directory":
		CreateDirectoryRoute(request, conn)
	case "/rename":
		RenameRoute(request, conn)
	default:
	}
}

func RenameRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := GetCookie(existingCookies, request)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
		return
	}

	urlParameters := GetUrlParameters(request)

	parameterOldPath, oldPathExists := urlParameters["old-path"]
	if !oldPathExists || strings.Contains(parameterOldPath, "..") {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
		return
	}

	oldPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterOldPath)

	parameterNewPath, newPathExists := urlParameters["new-path"]
	if !newPathExists || strings.Contains(parameterNewPath, "..") {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
		return
	}

	newPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterNewPath)

	err = os.Rename(oldPath, newPath)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
	} else {
		_ = sendResponse(conn, "200 OK", []byte("file renames"))
	}

}
