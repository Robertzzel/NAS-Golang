package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

func CreateDirectoryTable(path, urlPath string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", nil
	}

	body := "<table class=\"table\">"
	body += "<thead><tr><th>Nume</th><th>Marime</th><th>ACCES</th><th>DOWNLOAD</th><th>REMOVE</th></tr></thead><tbody>"
	for _, e := range entries {
		body += CreateTableRowFromEntry(e, urlPath)
	}
	body += "</tbody></table>"
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
		body += fmt.Sprintf("<td><a href=\"/display?path=%s\">ACCES</a></td>", downUrl)
		body += fmt.Sprintf("<td><a href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("/download?path=%s", downUrl))
	} else {
		body += fmt.Sprintf("<td>%s</td>", formatBytes(uint64(info.Size())))
		body += "<td>-</td>"
		body += fmt.Sprintf("<td><a href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("/download?path=%s", downUrl))
	}

	body += fmt.Sprintf("<td><a href=\"%s\">REMOVE</a></td>", fmt.Sprintf("/delete?path=%s", downUrl))

	body += fmt.Sprintf(`<td>
	<label for="filename%s">Enter Name:</label>
  	<input type="text" id="filename%s" name="filename" oninput="document.getElementById('filenameInput%s').href='/rename?old-path=%s&new-path=%s/' + encodeURIComponent(this.value)">
  	<a id="filenameInput%s" href="/rename?old-path=%s&new-path=%s/">Create</a></td>`, downUrl, downUrl, downUrl, downUrl, filepath.Dir(downUrl), downUrl, filepath.Dir(downUrl)) // create directory

	body += "</tr>"
	return body
}

func SendDirectoryStructure(conn *bufio.ReadWriter, path string, urlPath string) {
	body, err := CreateDirectoryTable(path, urlPath)
	if err != nil {
		_ = sendResponse(conn, "500 Server Erro", []byte("cannot display directory table"))
		return
	}

	body += fmt.Sprintf("<a href=\"/display?path=%s\">BACK</a>", filepath.Dir(urlPath))

	size, err := GetDirectorySize(UPLOAD_DIR)
	if err == nil {
		body += fmt.Sprintf("<p>Occupied memory: %s </p>", formatBytes(uint64(size)))
	} else {
		body += fmt.Sprintf("<p>Cannot display size: %s </p>", err.Error())
	}

	body += fmt.Sprintf(`<form enctype="multipart/form-data" action="/upload?path=%s" method="post"><label for="files">Select files:</label>
  <input type="file" id="files" name="files"  multiple><br><br>
  <input type="submit"></form>`, urlPath) // file upload

	directryBasePath := urlPath
	if directryBasePath == "/" {
		directryBasePath = ""
	}
	body += fmt.Sprintf(`<label for="dirname">Enter Directory Name:</label>
  <input type="text" id="dirname" name="dirname" placeholder="Enter path" oninput="document.getElementById('dynamicLink').href='/create-directory?path=%s/' + encodeURIComponent(this.value)">
  <a id="dynamicLink" href="/create-directory?path=%s/">Create</a>`, directryBasePath, directryBasePath) // create directory

	htmlPage := strings.ReplaceAll(TABLE_PAGE, "<%BODY%>", body)
	_, _ = conn.WriteString(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(htmlPage)))
	_, _ = conn.WriteString(htmlPage)
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
	_ = writer.Flush()
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

		inZipFile := path
		f, err := w.Create(inZipFile)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	filepath.Walk(inputDirectory, walker)
}
