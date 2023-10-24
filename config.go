package main

import (
	"strings"

	"github.com/urfave/cli/v2"
)

type config struct {
	onlyDegression     bool
	failWithoutResults bool
	threshold          float64
	base               string
	compare            []string
	benchCmd           string
	benchArgs          []string
	gitPath            string
}

func newConfig(c *cli.Context) config {
	return config{
		onlyDegression:     c.Bool("only-degression"),
		failWithoutResults: c.Bool("fail-without-results"),
		threshold:          c.Float64("threshold"),
		base:               c.String("base"),
		compare:            strings.Split(c.String("compare"), ","),
		benchCmd:           c.String("bench-cmd"),
		benchArgs:          strings.Fields(c.String("bench-args")),
		gitPath:            c.String("git-path"),
	}
}
