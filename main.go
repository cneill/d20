package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func runServer(ctx *cli.Context) error {
	serverOpts := &ServerOpts{
		Host: ctx.String("host"),
		Port: ctx.Int("port"),
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
