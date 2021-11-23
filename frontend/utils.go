package main

import (
	"log"
	"os"
)

// Setup the bidder by reading the environment variable BIDDER_NAME
func setupBidder() {
	bidderName = os.Getenv("BIDDER_NAME")

	if bidderName == "" {
		log.Fatalf("No bidder name provided")
	}
}
