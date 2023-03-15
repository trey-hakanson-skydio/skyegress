package main

import (
	"github.com/alecthomas/kong"
	"github.com/treyhaknson/skyegress/pkg/cmd"
)

type CLI struct {
	cmd.Common

	Serve  cmd.ServeCmd  `kong:"cmd,help='Start the skyegress server'"`
	Client cmd.ClientCmd `kong:"cmd,help='Issue a request to the server'"`
}

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli)
	ctx.Bind(&cli.Common)
	err := ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}
