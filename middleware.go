package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type contextKey string

const userKey contextKey = "USER"

func (s *Server) UserMiddleware(required bool, next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		dataCookie, err := req.Cookie(CookieData)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				if required {
					http.Redirect(writer, req, "/", http.StatusSeeOther)
					return
				}

				next(writer, req)

				return
			}

			s.doErr(writer, fmt.Sprintf("Failed to get cookie: %v", err))

			return
		}

		if dataCookie.Value == "" {
			s.logout(writer, req)
			return
		}

		user, err := UserFromCookie(dataCookie.Value, s.secretKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "user cookie error: %v\n", err)
		}

		if err != nil || user == nil {
			s.logout(writer, req)
			return
		}

		ctx := context.WithValue(req.Context(), userKey, user)
		req = req.WithContext(ctx)

		next(writer, req)
	}
}

func (s *Server) GameMasterMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		user := UserFromContext(req)

		if !user.IsGameMaster {
			s.doErr(writer, "YOU ARE NOT THE GAME MASTER")
			return
		}

		next(writer, req)
	}
}
