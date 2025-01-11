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
	"time"
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

	body := "<table style=\"border: 1px solid black;\">"
	body += "<tr><th>Nume</th><th>Marime</th><th>ACCES</th><th>DOWNLOAD</th></tr>"
	for _, e := range entries {
		body += CreateTableRowFromEntry(e, urlPath)
	}
	body += "</table>"
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

	body += "</tr>"
	return body
}

func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
		PB = TB * 1024
	)

	switch {
	case bytes >= PB:
		return fmt.Sprintf("%.2f PB", float64(bytes)/float64(PB))
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func GetDirectorySize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			size += f.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

func SendDirectoryStructure(conn *bufio.ReadWriter, path string, urlPath string) {
	htmlPage := "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><meta http-equiv=\"X-UA-Compatible\" content=\"ie=edge\"><title>HTML 5 Boilerplate</title></head><body><%BODY%></body></html>"

	body, err := CreateDirectoryTable(path, urlPath)
	if err != nil {
		_ = sendResponse(conn, "500 Server Erro", []byte("cannot display directory table"))
		return
	}

	body += fmt.Sprintf("<a href=\"/display?path=%s\">BACK</a>", filepath.Dir(urlPath))

	size, err := GetDirectorySize(UPLOAD_DIR)
	if err == nil {
		body += fmt.Sprintf("<p>Remaining memory: %s </p>", formatBytes(uint64(size)))
	} else {
		body += fmt.Sprintf("<p>Cannot display size: %s </p>", err.Error())
	}

	htmlPage = strings.ReplaceAll(htmlPage, "<%BODY%>", body)
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
	_, _ = conn.WriteString("Content-Length: " + strconv.FormatInt(fileSize, 10) + "\r\n\r\n")

	_ = conn.Flush()

	_, err = io.Copy(conn, file)
	if err != nil {
		_, _ = conn.WriteString("Error sending file content.")
	}
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
			_ = sendHTMLResponseWithHeaders(writer, "200 OK", []byte("success"), fmt.Sprintf("Set-Cookie: drive=%s", cookie.value))
		} else {
			_ = sendResponse(writer, "400 Bad Request", []byte("user not found"))
		}
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
