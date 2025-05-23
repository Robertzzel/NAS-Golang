package main

import (
	"errors"
	"net/http"
	"time"
)

type Cookie struct {
	username string
	value    string
	expires  time.Time
}

type CookieStore struct {
	cookies []*Cookie
}

func NewCookieStore() CookieStore { return CookieStore{cookies: []*Cookie{}} }

func (store *CookieStore) CreateCookie(username string) *Cookie {
	cookie := &Cookie{
		username: username,
		value:    generateRandomString(150),
		expires:  time.Now().Add(time.Hour * 24),
	}
	store.cookies = append(store.cookies, cookie)
	return cookie
}

func (store *CookieStore) GetCookie(request *http.Request) (*Cookie, error) {
	neededCookieValue := request.Header.Get("Cookie")
	if neededCookieValue == "" {
		return nil, errors.New("cookie not found")
	}

	store.cookies = Filter(store.cookies, func(cookie *Cookie) bool {
		return cookie.expires.After(time.Now())
	})

	cookie := FirstOr(store.cookies, func(cookie *Cookie) bool {
		return cookie.value == neededCookieValue
	}, nil)

	if cookie == nil {
		return nil, errors.New("cookie not found")
	}
	return cookie, nil
}
