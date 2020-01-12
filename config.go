package main

import "github.com/urfave/cli/v2"

type config struct {
	args           []string
	onlyDegression bool
	threshold      float64
	bench          string
	benchmem       bool
	benchtime      string
}

func newConfig(c *cli.Context) config {
	return config{
		args:           c.Args().Slice(),
		onlyDegression: c.Bool("only-degression"),
		threshold:      c.Float64("threshold"),
		bench:          c.String("bench"),
		benchmem:       c.Bool("benchmem"),
		benchtime:      c.String("benchtime"),
	}
}
