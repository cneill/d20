package main

import (
	"fmt"
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
		partyKey := req.Form.Get("party-key")

		if name == "" {
			s.doErr(writer, "You must enter a name.")
			return
		}

		if partyKey != s.Opts.Config.PartyKey {
			s.doErr(writer, "Invalid party key")
			return
		}

		var ipAddr string

		if forwarded := req.Header.Get("X-Forwarded-For"); forwarded != "" {
			ipAddr = maskIP(forwarded)
		} else {
			ipAddr = maskIP(req.RemoteAddr)
		}

		user := &User{
			Name:          name,
			CharacterName: characterName,
			IsGameMaster:  characterName == s.Opts.Config.GameMasterName,
			IPAddress:     ipAddr,
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
		Stats   *Stats
		OOB     bool
	}{
		User:    user,
		History: s.Rolls.Sort(),
		Stats:   s.Stats,
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

	critOn, err := strconv.ParseInt(req.Form.Get("crit-on"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid 'crit on' number: %v", err))
		return
	}

	complicationOn, err := strconv.ParseInt(req.Form.Get("complication-on"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid 'complication on' number: %v", err))
		return
	}

	dice := NewDice(int(sides), int(num), int(critOn), int(complicationOn))
	roll := dice.Roll(user)

	s.rollMutex.Lock()
	s.Rolls = append(s.Rolls, roll)
	s.rollMutex.Unlock()

	s.NotifyClients(EventTypeRoll)

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

func (s *Server) StatsHandler(writer http.ResponseWriter, req *http.Request) {
	if err := s.Renderer.ExecuteSingle(writer, "stats", s.Stats); err != nil {
		s.doErr(writer, fmt.Sprintf("failed to execute history template: %v", err))
		return
	}
}

func (s *Server) GameMasterHandler(writer http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		s.doErr(writer, fmt.Sprintf("failed to parse form: %v", err))
		return
	}

	threat, err := strconv.ParseInt(req.Form.Get("threat"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid number of threat: %v", err))
		return
	}

	momentum, err := strconv.ParseInt(req.Form.Get("momentum"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid number of momentum: %v", err))
		return
	}

	sceneTraitsRaw := req.Form.Get("scene-traits")
	sceneTraitParts := strings.Split(sceneTraitsRaw, ",")
	sceneTraits := make([]string, 0, len(sceneTraitParts))

	for _, sceneTrait := range sceneTraitParts {
		cleaned := strings.TrimSpace(sceneTrait)
		if cleaned != "" {
			sceneTraits = append(sceneTraits, cleaned)
		}
	}

	// characterTraitsRaw := req.Form.Get("character-traits")

	s.Stats.SetThreat(int(threat))
	s.Stats.SetMomentum(int(momentum))
	s.Stats.SetSceneTraits(sceneTraits)
	s.NotifyClients(EventTypeStats)
}

func (s *Server) PrivateRollHandler(writer http.ResponseWriter, req *http.Request) {
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

	critOn, err := strconv.ParseInt(req.Form.Get("crit-on"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid 'crit on' number: %v", err))
		return
	}

	complicationOn, err := strconv.ParseInt(req.Form.Get("complication-on"), 10, 64)
	if err != nil {
		s.doErr(writer, fmt.Sprintf("invalid 'complication on' number: %v", err))
		return
	}

	dice := NewDice(int(sides), int(num), int(critOn), int(complicationOn))
	roll := dice.Roll(user)

	if err := s.Renderer.ExecuteSingle(writer, "private_roll", roll); err != nil {
		s.doErr(writer, fmt.Sprintf("failed to execute history template: %v", err))
		return
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
