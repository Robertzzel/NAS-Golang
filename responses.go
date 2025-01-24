package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func createStringResponse(status int, body string) string {
	return fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		status, http.StatusText(status), len(body), body,
	)
}

func createHTMLResponse(status int, body string) string {
	return fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n%s",
		status, http.StatusText(status), len(body), body,
	)
}

func createHTMLResponseWithHeaders(status int, body string, headers []string) string {
	combinedHeaders := ""
	for _, header := range headers {
		combinedHeaders += fmt.Sprintf("%s\r\n", header)
	}
	combinedHeaders += "\r\n"

	return fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/html\r\nContent-Length: %d\r\n%s%s",
		status, http.StatusText(status), len(body), combinedHeaders, body,
	)
}

func sendResponse(conn *bufio.ReadWriter, status string, body []byte) error {
	header := fmt.Sprintf(
		"HTTP/1.1 %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n",
		status, len(body),
	)
	if _, err := conn.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := conn.Write(body); err != nil {
		return err
	}
	return nil
}

func sendHTMLResponse(conn *bufio.ReadWriter, status string, body []byte) error {
	header := fmt.Sprintf(
		"HTTP/1.1 %s\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n",
		status, len(body),
	)
	if _, err := conn.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := conn.Write(body); err != nil {
		return err
	}
	return nil
}

func sendHTMLResponseWithHeaders(conn *bufio.ReadWriter, status string, body []byte, headers string) error {
	header := fmt.Sprintf(
		"HTTP/1.1 %s\r\nContent-Type: text/html\r\nContent-Length: %d\r\n%s\r\n",
		status, len(body), headers,
	)
	if _, err := conn.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := conn.Write(body); err != nil {
		return err
	}
	return nil
}

func CreateDirectoryTable(path, urlPath, filter string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", nil
	}

	body := `<div class="table-container"><table class="table">`
	body += "<thead><tr><th>Nume</th><th>Marime</th><th>ACCES</th><th>DOWNLOAD</th><th>REMOVE</th><th>RENAME</th></tr></thead><tbody>"
	for _, e := range entries {
		if strings.Contains(e.Name(), filter) {
			body += CreateTableRowFromEntry(e, urlPath)
		}
	}
	body += "</tbody></table></div>"
	return body, nil
}

func CreateTableRowFromEntry(file os.DirEntry, dirName string) string {
	body := "<tr>"
	body += fmt.Sprintf("<td>%s</td>", file.Name())

	downUrl := fmt.Sprintf("%s/%s", dirName, file.Name())
	if strings.HasPrefix(downUrl, "//") {
		downUrl = downUrl[1:]
	}

	info, err := file.Info()
	if err != nil {
		body += "</tr>"
		return body
	}

	if info.IsDir() {
		body += "<td>-</td>" // no size
		body += fmt.Sprintf("<td><a class=\"btn\" href=\"/display?path=%s\">ACCES</a></td>", downUrl)
		body += fmt.Sprintf("<td><a class=\"btn\" href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("/download?path=%s", downUrl))
	} else {
		body += fmt.Sprintf("<td>%s</td>", formatBytes(uint64(info.Size())))
		body += "<td>-</td>"
		body += fmt.Sprintf("<td><a class=\"btn\" href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("/download?path=%s", downUrl))
	}

	body += fmt.Sprintf("<td><a class=\"btn\" class=\"btn\" href=\"%s\">REMOVE</a></td>", fmt.Sprintf("/delete?path=%s", downUrl))

	body += fmt.Sprintf(`<td>
	<label for="filename%s">Enter Name:</label>
  	<input type="text" id="filename%s" name="filename" oninput="document.getElementById('filenameInput%s').href='/rename?old-path=%s&new-path=%s/' + encodeURIComponent(this.value)">
  	<a class="btn" id="filenameInput%s" href="/rename?old-path=%s&new-path=%s/">Rename</a></td>`, downUrl, downUrl, downUrl, downUrl, filepath.Dir(downUrl), downUrl, filepath.Dir(downUrl)) // create directory

	body += "</tr>"
	return body
}

func CreateDirectoryStructure(path, urlPath, filter string) string {
	directryBasePath := urlPath
	if directryBasePath == "/" {
		directryBasePath = ""
	}

	body := fmt.Sprintf(`<div class="search">
        <input type="text" id="filter" name="filter" placeholder="Search" oninput="document.getElementById('filterA').href='/display?path=%s/&filter=' + encodeURIComponent(this.value)">
        <a class="btn" id="filterA" href="/display?path=%s/">Search</a>
    <div>`, directryBasePath, directryBasePath)

	tableHtml, err := CreateDirectoryTable(path, urlPath, filter)
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "cannot display directory table")
	}

	body += tableHtml
	body += fmt.Sprintf("<a class=\"btn\" href=\"/display?path=%s\">BACK</a>", filepath.Dir(urlPath))

	size, err := GetDirectorySize(UPLOAD_DIR)
	if err == nil {
		body += fmt.Sprintf("<p>Occupied memory: %s </p>", formatBytes(uint64(size)))
	} else {
		body += fmt.Sprintf("<p>Cannot display size: %s </p>", err.Error())
	}

	body += fmt.Sprintf(`<form  enctype="multipart/form-data" action="/upload?path=%s" method="post"><label>Select files for upload:</label>
  <input style="display:inline;" type="file" id="files" name="files" multiple>
  <input type="submit" value="Upload"></form>`, urlPath) // file upload

	body += fmt.Sprintf(`<div style="margin-top: 10px"><label for="dirname">Create directory:</label>
  <input type="text" id="dirname" name="dirname" placeholder="Enter path" oninput="document.getElementById('dynamicLink').href='/create-directory?path=%s/' + encodeURIComponent(this.value)">
  <a class="btn" id="dynamicLink" href="/create-directory?path=%s/">Create</a><div>`, directryBasePath, directryBasePath) // create directory

	html, err := GetTablePageHTML()
	if err != nil {
		return createStringResponse(http.StatusInternalServerError, "cannot display directory table")
	}

	htmlPage := strings.ReplaceAll(html, "<%BODY%>", body)
	return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n%s", len(htmlPage), htmlPage)
}

func SendFile(conn *bufio.ReadWriter, path string) {
	file, err := os.Open(path)
	if err != nil {
		_ = sendResponse(conn, "404 Not Found", []byte("File not found"))
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		_ = sendResponse(conn, "500 Internal Server Error", []byte("Could not retrieve file information"))
		return
	}
	fileSize := fileInfo.Size()

	_, _ = conn.WriteString("HTTP/1.1 200 OK\r\n")
	_, _ = conn.WriteString("Content-Type: application/octet-stream\r\n")
	_, _ = conn.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filepath.Base(file.Name())))
	_, _ = conn.WriteString("Content-Length: " + strconv.FormatInt(fileSize, 10) + "\r\n\r\n")

	_ = conn.Flush()

	_, err = io.Copy(conn, file)
	if err != nil {
		_, _ = conn.WriteString("Error sending file content.")
	}
}

func SendDirectoryAsZip(inputDirectory string, writer *bufio.ReadWriter) {
	_, _ = writer.WriteString("HTTP/1.1 200 OK\r\n")
	_, _ = writer.WriteString("Content-Type: application/octet-stream\r\n")
	_, _ = writer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s.zip\"\r\n\r\n", inputDirectory))

	w := zip.NewWriter(writer)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	_ = filepath.Walk(inputDirectory, walker)
}
