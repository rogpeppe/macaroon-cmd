package main

import (
	"os"

	"github.com/juju/cmd"
)

func main() {
	os.Exit(main1(os.Args))
}

func main1(args []string) int {
	c := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name: "macaroon",
		Log:  &cmd.Log{},
	})
	for _, subc := range registry {
		c.Register(subc)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return cmd.Main(c, &cmd.Context{
		Dir:    cwd,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}, os.Args[1:])
}

var registry []cmd.Command

func register(c cmd.Command) {
	registry = append(registry, c)
}
