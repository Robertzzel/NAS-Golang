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

func SendDirectoryStructure(conn *bufio.ReadWriter, path, trimmedUtlPath string) {
	htmlPage := "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><meta http-equiv=\"X-UA-Compatible\" content=\"ie=edge\"><title>HTML 5 Boilerplate</title></head><body><%BODY%></body></html>"

	entries, err := os.ReadDir(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}

	body := "<table style=\"border: 1px solid black;\">"
	body += "<tr><th>Nume</th><th>Marime</th><th>ACCES</th><th>DOWNLOAD</th></tr>"
	for _, e := range entries {
		body += "<tr>"

		body += fmt.Sprintf("<td>%s</td>", e.Name())

		info, err := e.Info()
		if err != nil {
			body += "</tr>"
			continue
		}

		downUrl := fmt.Sprintf("%s/%s", trimmedUtlPath, e.Name())
		if strings.HasPrefix(downUrl, "//") {
			downUrl = downUrl[1:]
		}

		if info.IsDir() {
			body += "<td>-</td>" // no size
			body += fmt.Sprintf("<td><a href=\"/display?path=%s\">ACCES</a></td>", downUrl)
			body += fmt.Sprintf("<td><a href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("/download?path=%s", downUrl))
		} else {
			body += fmt.Sprintf("<td>%d</td>", info.Size())
			body += "<td>-</td>"
			body += fmt.Sprintf("<td><a href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("/download?path=%s", downUrl))
		}

		body += "</tr>"
	}
	body += "</table>"
	body += fmt.Sprintf("<a href=\"/display?path=%s\">BACK</a>", filepath.Dir(trimmedUtlPath))

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
