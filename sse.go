package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type EventMessage struct {
	EventType EventType
	Data      []byte
}

type EventType string

func (e EventType) String() string { return string(e) }

const (
	EventTypeRoll  EventType = "ROLL"
	EventTypeStats EventType = "STATS"
)

func (s *Server) NotifyClients(eventType EventType) {
	var buf bytes.Buffer

	switch eventType {
	case EventTypeRoll:
		data := struct {
			History Rolls
			OOB     bool
		}{
			History: s.Rolls.Sort(),
			OOB:     false,
		}

		// Render the new row HTML
		if err := s.Renderer.ExecuteSingle(&buf, "history", data); err != nil {
			log.Printf("Error rendering history: %v", err)
			return
		}

	case EventTypeStats:
		data := s.Stats

		// Render the new row HTML
		if err := s.Renderer.ExecuteSingle(&buf, "stats", data); err != nil {
			log.Printf("Error rendering stats: %v", err)
			return
		}

	default:
		fmt.Fprintf(os.Stderr, "UNKNOWN EVENT TYPE: %s\n", eventType)
	}

	htmlData := struct {
		HTML string `json:"html"`
	}{
		HTML: buf.String(),
	}

	data, err := json.Marshal(htmlData)
	if err != nil {
		panic(fmt.Errorf("failed to encode JSON: %w", err))
	}

	message := EventMessage{
		EventType: eventType,
		Data:      data,
	}

	s.clientMutex.RLock()
	defer s.clientMutex.RUnlock()

	for clientChan := range s.clients {
		select {
		case clientChan <- message:
		default:
			// Client is not ready to receive the message, skip it
		}
	}
}

func (s *Server) SSEHandler(writer http.ResponseWriter, req *http.Request) {
	// Set headers for SSE
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	// Create a channel for this client
	messageChan := make(chan EventMessage)

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
			msg := fmt.Sprintf("event: %s\ndata: %s\n\n", message.EventType, message.Data)
			fmt.Printf("MESSAGE: %s\n", msg)
			fmt.Fprint(writer, msg)
			writer.(http.Flusher).Flush()
		case <-req.Context().Done():
			return
		}
	}
}
