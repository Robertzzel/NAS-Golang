package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func DownloadRoute(request *http.Request, conn *bufio.Writer) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterPath := request.URL.Query().Get("path")
	if parameterPath == "" || strings.Contains(parameterPath, "..") {
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

func UploadRoute(request *http.Request, conn *bufio.Writer) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	err = request.ParseMultipartForm(1 << 28)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
	}

	parameterPath := request.FormValue("path")
	if parameterPath == "" || strings.Contains(parameterPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	path := filepath.Join(UPLOAD_DIR, cookie.username, parameterPath)

	files := request.MultipartForm.File["files"]
	for _, fh := range files {
		if err := os.MkdirAll(path, 0755); err != nil {
			sendEmptyResponse(conn, http.StatusBadRequest)
			return
		}
		file := filepath.Join(path, fh.Filename)
		out, err := os.Create(file)
		if err != nil {
			sendEmptyResponse(conn, http.StatusBadRequest)
			return
		}
		defer out.Close()

		src, err := fh.Open()
		if err != nil {
			sendEmptyResponse(conn, http.StatusInternalServerError)
			return
		}
		defer src.Close()

		_, err = io.Copy(out, src)
		if err != nil {
			sendEmptyResponse(conn, http.StatusBadRequest)
			return
		}
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	sendEmptyResponse(conn, http.StatusOK)
}

func GetDirectoryStructureRoute(request *http.Request, conn *bufio.Writer) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusUnauthorized)
		return
	}

	parameterPath := request.URL.Query().Get("path")
	if parameterPath == "" || strings.Contains(parameterPath, "..") {
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

	json, err := CreateDirectoryJson(path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}

	sendJsonResponse(conn, http.StatusOK, json)
}

func LoginGetRoute(conn *bufio.Writer) {
	html, err := GetLoginPageHTML()
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
	} else {
		sendHTMLResponse(conn, http.StatusOK, html)
	}
}

func LoginPostRoute(request *http.Request, conn *bufio.Writer) {
	err := request.ParseForm()
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	users, err := ParseUsersFile(USERS_FILE)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}

	userFound := Any(users, func(user []string) bool {
		return user[0] == request.FormValue("username") && user[1] == request.FormValue("password")
	})

	if userFound {
		cookie := cookieStore.CreateCookie(request.FormValue("username"))
		header1 := fmt.Sprintf("Set-Cookie: %s", cookie.value)
		header2 := "Location: /home"
		sendEmptyResponseWithHeaders(conn, http.StatusFound, []string{header1, header2})
	} else {
		sendEmptyResponse(conn, http.StatusBadRequest)
	}
}

func DeleteRoute(request *http.Request, conn *bufio.Writer) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	form := ParseRequestBody(request.Body)
	if form == nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterPath := form["path"]
	if parameterPath == "" || strings.Contains(parameterPath, "..") {
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
		return
	}

	sendEmptyResponse(conn, http.StatusOK)
}

func CreateDirectoryRoute(request *http.Request, conn *bufio.Writer) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusUnauthorized)
		return
	}

	form := ParseRequestBody(request.Body)
	if form == nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterPath := form["path"]
	if parameterPath == "" || strings.Contains(parameterPath, "..") {
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

func RenameRoute(request *http.Request, conn *bufio.Writer) {
	cookie, err := cookieStore.GetCookie(request)
	if err != nil {
		sendEmptyResponse(conn, http.StatusUnauthorized)
		return
	}

	form := ParseRequestBody(request.Body)
	if form == nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	parameterOldPath := form["old"]
	if parameterOldPath == "" || strings.Contains(parameterOldPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}
	oldPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterOldPath)

	parameterNewPath := form["new"]
	if parameterNewPath == "" || strings.Contains(parameterNewPath, "..") {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}
	newPath := filepath.Join(UPLOAD_DIR, cookie.username, parameterNewPath)

	err = os.MkdirAll(filepath.Dir(newPath), 0755)
	if err != nil {
		sendEmptyResponse(conn, http.StatusBadRequest)
		return
	}

	err = os.Rename(oldPath, newPath)
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}
	sendEmptyResponse(conn, http.StatusOK)
}
