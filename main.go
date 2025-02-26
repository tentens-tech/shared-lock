package main

import (
	"context"
	"log"

	"github.com/tentens-tech/shared-lock/internal/delivery"
)

func main() {
	if err := delivery.Execute(context.Background()); err != nil {
		log.Fatal(err)
	}
}
