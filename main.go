package main

import (
	"log"
	"os"
	"tentens-tech/shared-lock/sharedLock"
)

func main() {
	err := sharedLock.StartSharedLock()
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(0)
}
