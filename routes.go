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
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterPath, pathExists := request.UrlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	info, err := os.Stat(path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
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
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterPath, pathExists := request.UrlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)
	err = handleMultipartBody(conn.Reader, request.Headers["Content-Type"], path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	sendEmptyResponse(conn, http.StatusOK)
}

func GetDirectoryStructureRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusUnauthorized)
		return
	}

	parameterPath, pathExists := request.UrlParameters["path"]
	if !pathExists {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}
	parameterPath = CleanFilePath(parameterPath)

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	info, err := os.Stat(path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	isFile := !info.IsDir()
	if isFile {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	filter := request.UrlParameters["filter"]

	json, err := CreateDirectoryJson(path, filter)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}

	sendJsonResponse(conn, http.StatusOK, json)
}

func LoginGetRoute(conn *bufio.ReadWriter) {
	html, err := GetLoginPageHTML()
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
	} else {
		sendHTMLResponse(conn, http.StatusOK, html)
	}
}

func LoginPostRoute(conn *bufio.ReadWriter) {
	bodyBytes := make([]byte, 1024)
	bytes, _ := conn.Read(bodyBytes)
	body := string(bodyBytes[:bytes])

	form := ParseFormBody(body)

	users, err := ParseUsersFile(USERS_FILE)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}

	userFound := Any(users, func(user []string) bool {
		return user[0] == form["username"] && user[1] == form["password"]
	})

	if userFound {
		cookie := cookieStore.CreateCookie(form["username"])
		header1 := fmt.Sprintf("Set-Cookie: drive=%s", cookie.value)
		header2 := "Location: /home"
		sendEmptyResponseWithHeaders(conn, http.StatusFound, []string{header1, header2})
	} else {
		sendEmptyResponse(conn, http.StatusBadRequest)
	}
}

func DeleteRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterPath, pathExists := request.UrlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	_, err = os.Stat(path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusNotFound)
		return
	}

	err = os.RemoveAll(path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
	}

	sendEmptyResponse(conn, http.StatusOK)
}

func CreateDirectoryRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusUnauthorized)
		return
	}

	parameterPath, pathExists := request.UrlParameters["path"]
	if !pathExists || strings.Contains(parameterPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	_, err = os.Stat(path)
	if !os.IsNotExist(err) {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	err = os.Mkdir(path, 0755)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}

	sendEmptyResponse(conn, http.StatusOK)
}

func RenameRoute(request *Request, conn *bufio.ReadWriter) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusUnauthorized)
		return
	}

	parameterOldPath, oldPathExists := request.UrlParameters["old"]
	if !oldPathExists || strings.Contains(parameterOldPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}
	oldPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterOldPath)

	parameterNewPath, newPathExists := request.UrlParameters["new"]
	if !newPathExists || strings.Contains(parameterNewPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}
	newPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterNewPath)

	err = os.Rename(oldPath, newPath)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
	}
	sendEmptyResponse(conn, http.StatusOK)
}
