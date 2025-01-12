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

var (
	HOST             = ""
	PORT             = ""
	UPLOAD_DIR       = ""
	USERS_FILE       = ""
	CERTIFICATE_FILE = ""
	KEY_FILE         = ""
)

var existingCookies = make([]Cookie, 0)

func main() {
	if len(os.Args) != 7 {
		fmt.Printf("Usage: %s HOST PORT UPLOAD_DIR USERS_FILE CERTIFICATE KEY\n", os.Args[0])
		return
	}

	HOST = os.Args[1]
	PORT = os.Args[2]
	UPLOAD_DIR = os.Args[3]
	USERS_FILE = os.Args[4]
	CERTIFICATE_FILE = os.Args[5]
	KEY_FILE = os.Args[6]

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

	switch urlPath {
	case "/download":
		_, err := GetCookie(existingCookies, request)
		if err != nil {
			_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
			return
		}
		DownloadRoute(request, conn)
	case "/upload":
		_, err := GetCookie(existingCookies, request)
		if err != nil {
			_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
			return
		}
		UploadRoute(request, conn)
	case "/display":
		_, err := GetCookie(existingCookies, request)
		if err != nil {
			_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
			return
		}
		DisplayRoute(request, conn)
	case "/delete":
		_, err := GetCookie(existingCookies, request)
		if err != nil {
			_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
			return
		}
		DeleteRoute(request, conn)
	case "/create-directory":
		_, err := GetCookie(existingCookies, request)
		if err != nil {
			_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
			return
		}
		CreateDirectoryRoute(request, conn)
	case "/rename":
		_, err := GetCookie(existingCookies, request)
		if err != nil {
			_ = sendResponse(conn, "400 Bad Request", []byte("Not logged in"))
			return
		}
		RenameRoute(request, conn)
	default:
	}
}
