package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	sb := strings.Builder{}
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(charset[seededRand.Intn(len(charset))])
	}
	return sb.String()
}

func ParseUsersFile(path string) ([][]string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(file)
	result := make([][]string, 0)

	for _, line := range strings.Split(text, "\n") {
		parts := strings.Split(line, ",")
		if len(parts) == 2 {
			result = append(result, parts)
		}
	}

	return result, nil
}

func GetLoginPageHTML() (string, error) {
	file, err := os.ReadFile("./static/login.html")
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func GetTablePageHTML() (string, error) {
	file, err := os.ReadFile("./static/table.html")
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func Filter[T any](in []T, f func(T) bool) []T {
	newL := make([]T, 0)
	for _, v := range in {
		if f(v) {
			newL = append(newL, v)
		}
	}
	return newL
}

func Any[T any](in []T, f func(T) bool) bool {
	for _, v := range in {
		if f(v) {
			return true
		}
	}
	return false
}

func FirstOr[T any](in []T, f func(T) bool, def T) T {
	for _, v := range in {
		if f(v) {
			return v
		}
	}
	return def
}
