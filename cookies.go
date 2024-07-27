package main

import (
	"net/http"
	"time"
)

const CookieData = "data"

func (s *Server) resetCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:    name,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
	}
}
