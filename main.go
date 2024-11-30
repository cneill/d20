package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func runServer(ctx *cli.Context) error {
	configFile := ctx.String("config")

	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("invalid config file %q: %w", configFile, err)
	}

	config := &Config{}
	if err := json.Unmarshal(configBytes, config); err != nil {
		return fmt.Errorf("failed to parse config file %q: %w", configFile, err)
	}

	if err := config.OK(); err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	serverOpts := &ServerOpts{
		Host:   ctx.String("host"),
		Port:   ctx.Int("port"),
		Config: config,
	}

	server, err := NewServer(serverOpts)
	if err != nil {
		return fmt.Errorf("failed to set up server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func setup() error {
	app := &cli.App{
		Name:     "d20",
		HelpName: "d20",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start the server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Value: "0.0.0.0",
					},
					&cli.IntFlag{
						Name:  "port",
						Value: 8080,
					},
					&cli.StringFlag{
						Name:     "config",
						Value:    "config.json",
						Required: true,
					},
				},
				Action: runServer,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		return fmt.Errorf("error: %w", err)
	}

	return nil
}

func main() {
	if err := setup(); err != nil {
		panic(err)
	}
}
