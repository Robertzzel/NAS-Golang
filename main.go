package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
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

var cookieStore = NewCookieStore()
var bruteForceGuard = NewBruteForceGuard()

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
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			err := os.MkdirAll(path, 0755)
			if err != nil {
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

		if bruteForceGuard.IsBruteForceAttempt(conn.RemoteAddr()) {
			return
		}

		handleRequest(&request, readWriter)

		_ = readWriter.Flush()
		_ = conn.Close()
	}
}

func handleRequest(request *Request, conn *bufio.ReadWriter) {
	urlPath := GetUrlPath(request)

	if strings.Contains(urlPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	if urlPath == "/log" {
		if request.Method == "GET" {
			LoginGetRoute(conn)
			return
		} else if request.Method == "POST" {
			LoginPostRoute(conn)
			return
		}
	}

	_, err := cookieStore.GetCookie(request)
	if err != nil {
		return
	}

	switch true {
	case urlPath == "/home" && request.Method == "GET":
		page, _ := GetHomePageHTML()
		_, _ = conn.WriteString(page)
	case urlPath == "/download" && request.Method == "GET":
		DownloadRoute(request, conn)
	case urlPath == "/upload" && request.Method == "POST":
		UploadRoute(request, conn)
	case urlPath == "/directory" && request.Method == "GET":
		GetDirectoryStructureRoute(request, conn)
	case urlPath == "/delete" && request.Method == "GET":
		DeleteRoute(request, conn)
	case urlPath == "/create-directory" && request.Method == "GET":
		CreateDirectoryRoute(request, conn)
	case urlPath == "/rename" && request.Method == "GET":
		RenameRoute(request, conn)
	default:
	}
}
