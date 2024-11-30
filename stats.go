package main

import (
	"strings"
	"sync"
)

type SceneTraits []string

func (s SceneTraits) AsString() string {
	return strings.Join(s, ", ")
}

type Stats struct {
	Momentum        int                 `json:"momentum"`
	Threat          int                 `json:"threat"`
	SceneTraits     SceneTraits         `json:"scene_traits"`
	CharacterTraits map[string][]string `json:"character_traits"`

	Mutex sync.RWMutex
}

func (s *Stats) SetMomentum(value int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Momentum = value
}

func (s *Stats) SetThreat(value int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Threat = value
}

func (s *Stats) SetSceneTraits(value []string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.SceneTraits = value
}

func (s *Stats) SetCharacterTraits(character string, traits []string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.CharacterTraits[character] = traits

	if len(traits) == 0 {
		delete(s.CharacterTraits, character)
	}
}
