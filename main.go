package main

import (
	"log"
	"os"
	"path"
	"time"

	"github.com/keep-network/keep-tecdsa/cmd"
	"github.com/urfave/cli"
)

const (
	defaultConfigPath   = "./configs/config.toml"
	defaultBroadcastAPI = "blockcypher"
)

var (
	configPath   string
	broadcastAPI string
)

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "CLI for t-ECDSA keep"
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		{
			Name:  "Keep Network",
			Email: "info@keep.network",
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config,c",
			Value:       defaultConfigPath,
			Destination: &configPath,
			Usage:       "full path to the configuration file",
		},
		cli.StringFlag{
			Name:        "broadcast-api",
			Value:       defaultBroadcastAPI,
			Destination: &broadcastAPI,
			Usage:       "external service used to communicate with the blockchain",
		},
	}
	app.Commands = []cli.Command{
		cmd.SignCommand,
		cmd.StartCommand,
		cmd.PublishCommand,
		cmd.SmokeTestCommand,
	}

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}