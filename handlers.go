package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func (s *Server) IndexHandler(writer http.ResponseWriter, req *http.Request) {
	user := UserFromContext(req)
	if user != nil {
		http.Redirect(writer, req, "/dice", http.StatusSeeOther)
		return
	}

	switch req.Method {
	case http.MethodGet:
		if err := s.Renderer.ExecutePage(writer, "index", struct{}{}); err != nil {
			s.doErr(writer, fmt.Sprintf("Failed to execute index template: %s", err))
			return
		}

	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			s.doErr(writer, fmt.Sprintf("Failed to parse form: %v", err))
			return
		}

		name := req.Form.Get("name")
		characterName := req.Form.Get("character-name")

		if name == "" {
			s.doErr(writer, "You must enter a name.")
			return
		}

		user := &User{
			Name:          name,
			CharacterName: characterName,
			IsGameMaster:  characterName == "GM",
			IPAddress:     maskIP(req.RemoteAddr),
		}

		dataCookie, err := user.DataCookie(s.secretKey)
		if err != nil {
			s.doErr(writer, fmt.Sprintf("failed to save cookie: %v", err))
			return
		}

		http.SetCookie(writer, dataCookie)
		http.Redirect(writer, req, "/dice", http.StatusSeeOther)
	default:
		s.doErr(writer, fmt.Sprintf("Invalid HTTP method: %q", req.Method))
		return
	}
}

func maskIP(input string) string {
	parts := strings.Split(input, ".")
	parts[3] = "x"

	return strings.Join(parts, ".")
}

func (s *Server) DiceHandler(writer http.ResponseWriter, req *http.Request) {
	user := UserFromContext(req)

	data := struct {
		User    *User
		History Rolls
		OOB     bool
	}{
		User:    user,
		History: s.Rolls.Sort(),
		OOB:     true,
	}

	if err := s.Renderer.ExecutePage(writer, "dice", data); err != nil {
		s.doErr(writer, fmt.Sprintf("Failed to execute dice template: %v", err))
		return
	}
}

func (s *Server) RollHandler(writer http.ResponseWriter, req *http.Request) {
	user := UserFromContext(req)

	if err := req.ParseForm(); err != nil {
		s.doErr(writer, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	sides, err := strconv.ParseInt(req.Form.Get("sides"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid number of sides: %v", err))
		return
	}

	num, err := strconv.ParseInt(req.Form.Get("num"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid number of dice: %v", err))
		return
	}

	dice := NewDice(int(sides), int(num))
	roll := dice.Roll(user)

	s.rollMutex.Lock()
	s.Rolls = append(s.Rolls, roll)
	s.rollMutex.Unlock()

	s.NotifyClients()

	data := struct {
		User    *User
		History Rolls
		OOB     bool
	}{
		User:    user,
		History: s.Rolls.Sort(),
		OOB:     false,
	}

	if err := s.Renderer.ExecuteSingle(writer, "history", data); err != nil {
		s.doErr(writer, fmt.Sprintf("failed to execute history template: %v", err))
		return
	}
}

func (s *Server) NotifyClients() {
	data := struct {
		History Rolls
		OOB     bool
	}{
		History: s.Rolls.Sort(),
		OOB:     false,
	}

	// Render the new row HTML
	var buf bytes.Buffer
	if err := s.Renderer.ExecuteSingle(&buf, "history", data); err != nil {
		log.Printf("Error rendering history row: %v", err)
		return
	}

	message := struct {
		HTML string `json:"html"`
	}{
		HTML: buf.String(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		panic(fmt.Errorf("failed to encode JSON: %w", err))
	}

	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	for clientChan := range s.clients {
		select {
		case clientChan <- string(messageBytes):
		default:
			// Client is not ready to receive the message, skip it
		}
	}
}

func (s *Server) HistoryHandler(writer http.ResponseWriter, req *http.Request) {
	data := struct {
		History Rolls
		OOB     bool
	}{
		History: s.Rolls.Sort(),
		OOB:     false,
	}

	if err := s.Renderer.ExecuteSingle(writer, "history", data); err != nil {
		s.doErr(writer, fmt.Sprintf("failed to execute history template: %v", err))
		return
	}
}

func (s *Server) SSEHandler(writer http.ResponseWriter, req *http.Request) {
	// Set headers for SSE
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	// Create a channel for this client
	messageChan := make(chan string)

	// Register the client
	s.clientMutex.Lock()
	s.clients[messageChan] = true
	s.clientMutex.Unlock()

	// Remove the client when the connection is closed
	defer func() {
		s.clientMutex.Lock()
		delete(s.clients, messageChan)
		s.clientMutex.Unlock()
	}()

	for {
		select {
		case message := <-messageChan:
			msg := fmt.Sprintf("event: message\ndata: %s\n\n", message)
			fmt.Printf("MESSAGE: %s\n", msg)
			fmt.Fprint(writer, msg)
			writer.(http.Flusher).Flush()
		case <-req.Context().Done():
			return
		}
	}
}

func (s *Server) doErr(writer http.ResponseWriter, message string) {
	writer.WriteHeader(http.StatusInternalServerError)

	if _, writeErr := writer.Write([]byte(fmt.Sprintf("ERROR: %s\n", message))); writeErr != nil {
		fmt.Fprintf(os.Stderr, "failed to write error response: %v\noriginal error: %v\n", writeErr, message)
	}
}

func (s *Server) logout(writer http.ResponseWriter, req *http.Request) {
	http.SetCookie(writer, s.resetCookie(CookieData))
	http.Redirect(writer, req, "/", http.StatusSeeOther)
}
