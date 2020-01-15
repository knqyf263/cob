package main

import (
	"strings"

	"github.com/urfave/cli/v2"
)

type config struct {
	onlyDegression bool
	threshold      float64
	base           string
	benchCmd       string
	benchArgs      []string
}

func newConfig(c *cli.Context) config {
	return config{
		onlyDegression: c.Bool("only-degression"),
		threshold:      c.Float64("threshold"),
		base:           c.String("base"),
		benchCmd:       c.String("bench-cmd"),
		benchArgs:      strings.Fields(c.String("bench-args")),
	}
}
