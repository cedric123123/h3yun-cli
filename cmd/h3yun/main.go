package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/cedric123123/h3yun-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Getenv))
}
