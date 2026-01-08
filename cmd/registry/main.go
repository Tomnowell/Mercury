package main

import (
	"log"
)

func main() {
	err := NewServer(":5678")

	if err != nil {
		log.Fatal(err)
	}
}
