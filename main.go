package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
	"golang.org/x/tools/benchmark/parse"
)

func main() {
	app := &cli.App{
		Name:  "cob",
		Usage: "Continuous Benchmark for Go project",
		Action: func(c *cli.Context) error {
			return run(c)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	args := []string{"test", "-bench"}
	args = append(args, c.Args().Slice()...)
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		return err
	}

	b := bytes.NewBuffer(out)
	s, err := parse.ParseSet(b)
	if err != nil {
		return err
	}
	fmt.Println(s)
	return err
}
