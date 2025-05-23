package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func sendEmptyResponse(conn *bufio.Writer, status int) {
	header := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: 0\r\n\r\n",
		status, http.StatusText(status),
	)
	_, _ = conn.Write([]byte(header))
}

func sendEmptyResponseWithHeaders(conn *bufio.Writer, status int, headers []string) {
	combinedHeaders := ""
	for _, header := range headers {
		combinedHeaders += fmt.Sprintf("%s\r\n", header)
	}
	combinedHeaders += "\r\n"
	header := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: 0\r\n%s\r\n",
		status, http.StatusText(status), combinedHeaders,
	)
	_, _ = conn.Write([]byte(header))
}

func sendJsonResponse(conn *bufio.Writer, status int, json []byte) {
	header := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n",
		status, http.StatusText(status), len(json),
	)
	_, _ = conn.Write([]byte(header))
	_, _ = conn.Write(json)
}

func sendHTMLResponse(conn *bufio.Writer, status int, body string) {
	header := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n",
		status, http.StatusText(status), len(body),
	)
	_, _ = conn.Write([]byte(header))
	_, _ = conn.Write([]byte(body))
}

func Redirect(conn *bufio.Writer, to string) {
	sendEmptyResponseWithHeaders(conn, http.StatusFound, []string{fmt.Sprintf("Location: %s", to)})
}

func CreateDirectoryJson(path string) ([]byte, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return []byte{}, err
	}

	files := Map(entries, func(entry os.DirEntry) FileJson {
		return FileJsonFromDirEntry(entry)
	})
	marsh, err := json.Marshal(files)
	return marsh, err
}

type FileJson struct {
	Name        string
	IsDirectory bool
	Size        int64
}

func FileJsonFromDirEntry(file os.DirEntry) FileJson {
	info, err := file.Info()
	if err != nil {
		return FileJson{IsDirectory: file.IsDir(), Name: file.Name()}
	}
	return FileJson{IsDirectory: file.IsDir(), Name: file.Name(), Size: info.Size()}
}

func SendFile(conn *bufio.Writer, path string) {
	file, err := os.Open(path)
	if err != nil {
		sendEmptyResponse(conn, http.StatusNotFound)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		sendEmptyResponse(conn, http.StatusInternalServerError)
		return
	}
	fileSize := fileInfo.Size()

	_, _ = conn.WriteString("HTTP/1.1 200 OK\r\n")
	_, _ = conn.WriteString("Content-Type: application/octet-stream\r\n")
	_, _ = conn.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filepath.Base(file.Name())))
	_, _ = conn.WriteString("Content-Length: " + strconv.FormatInt(fileSize, 10) + "\r\n\r\n")
	_ = conn.Flush()
	_, _ = io.Copy(conn, file)
}

func SendDirectoryAsZip(inputDirectory string, writer *bufio.Writer) {
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
