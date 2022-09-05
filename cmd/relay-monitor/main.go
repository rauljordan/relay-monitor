package main

import (
	"context"
	"flag"
	"time"

	"github.com/ralexstokes/relay-monitor/internal/consensus"
)

var (
	configFile = flag.String("config", "config.example.yaml", "path to config file")
)

func main() {
	flag.Parse()
	bw, err := consensus.New(
		consensus.WithEndpoint("http://localhost:3500"),
		consensus.WithGenesisTime(1606824023),
	)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	bw.Start(ctx)
}
