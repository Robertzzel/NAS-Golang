package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
}

const (
	HOST       = "localhost"
	PORT       = "8080"
	UPLOAD_DIR = "./uploads"
)

func main() {
	if _, err := os.Stat(UPLOAD_DIR); os.IsNotExist(err) {
		if err := os.Mkdir(UPLOAD_DIR, 0755); err != nil {
			fmt.Printf("Error creating upload directory: %v\n", err)
			return
		}
	}

	listener, err := net.Listen("tcp", HOST+":"+PORT)
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

		readWriter := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		request, err := parseRequest(readWriter)
		if err != nil {
			_ = conn.Close()
			continue
		}

		defaultRoute(request, readWriter)

		_ = readWriter.Flush()
		_ = conn.Close()
	}

}

func GetUrlPath(url string) string { return strings.Split(url, "?")[0] }
func GetUrlParameters(url string) map[string]string {
	result := make(map[string]string)

	urlParts := strings.Split(url, "?")
	if len(urlParts) != 2 {
		return result
	}

	for _, param := range strings.Split(urlParts[1], "&") {
		paramParts := strings.Split(param, "=")
		if len(paramParts) == 2 {
			result[paramParts[0]] = paramParts[1]
		}
	}

	return result
}

func SendDirectoryAsZip(inputDirectory string, writer *bufio.ReadWriter) error {
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
	return filepath.Walk(inputDirectory, walker)
}

func defaultRoute(request Request, conn *bufio.ReadWriter) {
	path := GetUrlPath(request.Path)
	path = filepath.Join(UPLOAD_DIR, path)
	urlParameters := GetUrlParameters(request.Path)

	if strings.Contains(path, "..") {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}

	toDownload, toDownloadExists := urlParameters["download"]

	if info.IsDir() && toDownloadExists && toDownload == "true" {
		if err := SendDirectoryAsZip(path, conn); err != nil {
			_ = sendResponse(conn, "500 Server Error", []byte("cannot send directory"+err.Error()))
			return
		}
		return
	}
	if info.IsDir() && (toDownloadExists == false || toDownload == "false") {
		SendDirectoryStructure(conn, path, path)
		return
	}

	SendFile(conn, path)
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

func SendDirectoryStructure(conn *bufio.ReadWriter, path, trimmedUtlPath string) {
	htmlPage := "<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><meta http-equiv=\"X-UA-Compatible\" content=\"ie=edge\"><title>HTML 5 Boilerplate</title></head><body><%BODY%></body></html>"

	entries, err := os.ReadDir(path)
	if err != nil {
		_ = sendResponse(conn, "400 Bad Request", []byte("Bad Request"))
		return
	}

	body := "<table>"
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
			body += "<td>-</td>"
			body += fmt.Sprintf("<td><a href=\"%s\">DOWNLOAD</a></td>", fmt.Sprintf("%s?download=true", downUrl))
		} else {
			body += fmt.Sprintf("<td>%d</td>", info.Size())
		}

		body += fmt.Sprintf("<td><form method=\"get\" action=\"%s\"><button>ACCESS</button></form></td>", downUrl)

		body += "</tr>"
	}
	body += "</table>"

	htmlPage = strings.ReplaceAll(htmlPage, "<%BODY%>", body)
	_, _ = conn.WriteString(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(htmlPage)))
	_, _ = conn.WriteString(htmlPage)
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

func parseRequest(reader *bufio.ReadWriter) (Request, error) {
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return Request{}, err
	}

	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) < 3 {
		return Request{}, errors.New("less than 3 http params")
	}

	request := Request{Method: parts[0], Path: parts[1], Version: parts[2]}

	headers, err := parseHeaders(reader)
	if err != nil {
		return Request{}, nil
	}
	request.Headers = headers
	return request, nil
}

func parseHeaders(reader *bufio.ReadWriter) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return headers, nil
}
