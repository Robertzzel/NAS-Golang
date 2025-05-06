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
	log.Println("Starting with ", HOST, PORT, UPLOAD_DIR, USERS_FILE, CERTIFICATE_FILE, KEY_FILE)

	log.Println("Parsing users file...")
	users, err := ParseUsersFile(USERS_FILE)
	if err != nil {
		return
	}

	log.Print("Creating directory for users")
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

	log.Println("Reading SSL files...")
	cert, err := tls.LoadX509KeyPair(CERTIFICATE_FILE, KEY_FILE)
	if err != nil {
		log.Fatal("Error loading certificate. ", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	log.Println("Starting listening at", HOST+":"+PORT, "...")
	listener, err := tls.Listen("tcp", HOST+":"+PORT, tlsCfg)
	if err != nil {
		log.Println("Cannot listen,", err.Error())
		return
	}
	defer listener.Close()

	for {
		log.Println("Waiting for connection...")
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		log.Println("Accepted connection from", conn.RemoteAddr())
		readWriter := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		log.Println("Parsing request...")
		request, err := ParseRequest(readWriter)
		if err != nil {
			log.Println("Request could not be parsed, closing...")
			_ = conn.Close()
			continue
		}

		log.Println("Checking for bruteforce attempt...")
		if bruteForceGuard.IsBruteForceAttempt(conn.RemoteAddr()) {
			log.Println("Bruteforce detected...")
			return
		}

		log.Println("Handleing request...")
		HandleRequest(&request, readWriter)

		log.Println("Flushing and closing connection...")
		_ = readWriter.Flush()
		_ = conn.Close()
	}
}

func HandleRequest(request *Request, conn *bufio.ReadWriter) {
	urlPath := GetUrlPath(request)

	if strings.Contains(urlPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	if urlPath == "/log" {
		if request.Method == "GET" {
			log.Println("Login get route...")
			LoginGetRoute(conn)
			return
		} else if request.Method == "POST" {
			log.Println("Login post route...")
			LoginPostRoute(conn)
			return
		}
	}

	_, err := cookieStore.GetCookie(request)
	if err != nil {
		Redirect(conn, "/log")
		return
	}

	switch true {
	case urlPath == "/home" && request.Method == "GET":
		page, _ := GetHomePageHTML()
		sendHTMLResponse(conn, http.StatusOK, page)
	case urlPath == "/download" && request.Method == "GET":
		DownloadRoute(request, conn)
	case urlPath == "/upload" && request.Method == "POST":
		UploadRoute(request, conn)
	case urlPath == "/directory" && request.Method == "GET":
		log.Println("Getting file structure...")
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
