package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func DownloadRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := cookieStore.GetCookie(request)
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
	cookie, err := cookieStore.GetCookie(request)
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

func DisplayRoute(request *Request) string {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		return createStringResponse(http.StatusBadRequest, "Not logged in")
	}

	urlParameters := GetUrlParameters(request)

	parameterPath, pathExists := urlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		return createStringResponse(http.StatusBadRequest, "bad path")
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	info, err := os.Stat(path)
	if err != nil {
		return createStringResponse(http.StatusBadRequest, "Bad Request")
	}

	if !info.IsDir() {
		return createStringResponse(http.StatusBadRequest, "path not found")
	}

	filter := urlParameters["filter"]

	return CreateDirectoryStructure(path, parameterPath, filter)
}

func LoginGetRoute() string {
	html, err := GetLoginPageHTML()
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "Internal Server Error")
	}
	return createHTMLResponse(http.StatusOK, html)
}

func LoginPostRoute(body string) string {
	form := ParseFormBody(body)

	users, err := ParseUsersFile(USERS_FILE)
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "cannot parse users file")
	}

	found := Any(users, func(user []string) bool {
		return user[0] == form["username"] && user[1] == form["password"]
	})

	if found {
		cookie := cookieStore.CreateCookie(form["username"])
		return createHTMLResponseWithHeaders(http.StatusFound, "success", []string{fmt.Sprintf("Set-Cookie: drive=%s", cookie.value), "Location: /display?path=/"})
	} else {
		return createStringResponse(http.StatusBadRequest, "user not found")
	}
}

func DeleteRoute(request *Request) string {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		return createStringResponse(http.StatusBadRequest, "Not logged in")
	}

	urlParameters := GetUrlParameters(request)

	parameterPath, pathExists := urlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		return createStringResponse(http.StatusBadRequest, "bad path")
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	_, err = os.Stat(path)
	if err != nil {
		return createStringResponse(http.StatusNotFound, "path not found")
	}

	err = os.RemoveAll(path)
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "cannot delete path")
	}

	return createStringResponse(http.StatusOK, "success")
}

func CreateDirectoryRoute(request *Request) string {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		return createStringResponse(http.StatusBadRequest, "Not logged in")
	}

	urlParameters := GetUrlParameters(request)

	parameterPath, pathExists := urlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		return createStringResponse(http.StatusBadRequest, "bad path")
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	_, err = os.Stat(path)
	if !os.IsNotExist(err) {
		return createStringResponse(http.StatusBadRequest, "path already exists")
	}

	err = os.Mkdir(path, 0755)
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "cannot create directory")
	}

	return createStringResponse(http.StatusCreated, "directory created")
}

func RenameRoute(request *Request) string {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		return createStringResponse(http.StatusBadRequest, "Not logged in")
	}

	urlParameters := GetUrlParameters(request)

	parameterOldPath, oldPathExists := urlParameters["old-path"]
	if !oldPathExists || strings.Contains(parameterOldPath, "..") {
		return createStringResponse(http.StatusBadRequest, "bad path")
	}

	oldPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterOldPath)

	parameterNewPath, newPathExists := urlParameters["new-path"]
	if !newPathExists || strings.Contains(parameterNewPath, "..") {
		return createStringResponse(http.StatusBadRequest, "bad path")
	}

	newPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterNewPath)

	err = os.Rename(oldPath, newPath)
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "cannot rename file")
	}
	return createStringResponse(http.StatusOK, "success")
}
