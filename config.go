package main

import "fmt"

type Config struct {
	GameMasterName string `json:"game_master_name"`
	PartyKey       string `json:"party_key"`
}

func (c *Config) OK() error {
	if c.GameMasterName == "" {
		return fmt.Errorf("must supply game master name")
	}

	if c.PartyKey == "" {
		return fmt.Errorf("must supply party key")
	}

	return nil
}
