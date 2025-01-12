package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func UploadRoute(request *Request, conn *bufio.ReadWriter) {
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
	err = handleMultipartBody(conn.Reader, request.Headers["Content-Type"], path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Didn't receive the files"))
		return
	}

	_ = sendResponse(conn, "200 Ok", []byte("files uploaded"))
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

func LoginRoute(request *Request, writer *bufio.ReadWriter) {
	if request.Method == "GET" {
		_ = sendHTMLResponse(writer, "200 OK", []byte(LOGIN_FROM))
		return
	} else if request.Method == "POST" {
		body := make([]byte, 1024)
		bytes, _ := writer.Read(body)

		form := ParseFormBody(string(body[:bytes]))
		users, err := ParseUsersFile(USERS_FILE)
		if err != nil {
			_ = sendResponse(writer, "500 Server Error", []byte("cannot parse users file"))
			return
		}

		found := false
		for _, user := range users {
			if user[0] == form["username"] && user[1] == form["password"] {
				found = true
			}
		}

		if found {
			cookie := Cookie{
				username: form["username"],
				value:    generateRandomString(150),
				expires:  time.Now().Add(time.Hour * 24),
			}
			existingCookies = append(existingCookies, cookie)
			_ = sendHTMLResponseWithHeaders(writer, "302 Found", []byte("success"), fmt.Sprintf("Set-Cookie: drive=%s\r\nLocation: /display?path=/", cookie.value))
		} else {
			_ = sendResponse(writer, "400 Bad Request", []byte("user not found"))
		}
	}
}

func DeleteRoute(request *Request, conn *bufio.ReadWriter) {
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

	_, err = os.Stat(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
		return
	}

	err = os.RemoveAll(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad path"))
		return
	}

	_ = sendResponse(conn, "200 Ok", []byte("file removed"))
}

func CreateDirectoryRoute(request *Request, conn *bufio.ReadWriter) {
	if request.Method != "GET" {
		_ = sendResponse(conn, "400 Bad Request", []byte("bad request"))
		return
	}

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

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			_ = sendResponse(conn, "500 Server Erro", []byte("cannot create directory"))
		} else {
			_ = sendResponse(conn, "200 Ok", []byte("directory created"))
		}
	} else {
		_ = sendResponse(conn, "400 Bad Request", []byte("directory already exists"))
	}
}
