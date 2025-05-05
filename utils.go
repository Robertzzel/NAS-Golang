package main

import (
	"math/rand"
	"os"
	"strings"
	"time"
)

//func GetDirectorySize(dir string) (int64, error) {
//	var size int64
//	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
//		if !f.IsDir() {
//			size += f.Size()
//		}
//		return nil
//	})
//	if err != nil {
//		return 0, err
//	}
//	return size, nil
//}

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

func CleanFilePath(path string) string {
	return strings.ReplaceAll(path, "..", ".")
}

func GetLoginPageHTML() (string, error) {
	file, err := os.ReadFile("./static/login.html")
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func GetHomePageHTML() (string, error) {
	file, err := os.ReadFile("./static/home.html")
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func Map[A any, B any](in []A, f func(A) B) []B {
	newL := make([]B, 0, len(in))
	for _, v := range in {
		newL = append(newL, f(v))
	}
	return newL
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
