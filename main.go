package main

import (
	"context"
	"log"

	"github.com/tentens-tech/shared-lock/internal/delivery/cli"
)

func main() {
	if err := cli.Execute(context.Background()); err != nil {
		log.Fatal(err)
	}
}
