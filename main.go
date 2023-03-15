package main

import (
	"github.com/alecthomas/kong"
	"github.com/treyhaknson/skyegress/pkg/cmd"
	"github.com/treyhaknson/skyegress/pkg/config"
)

type CLI struct {
	config.Config

	Serve  cmd.ServeCmd  `kong:"cmd,help='Start the skyegress server'"`
	Client cmd.ClientCmd `kong:"cmd,help='Issue a request to the server'"`
}

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli)
	ctx.Bind(&cli.Config)
	err := ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}
