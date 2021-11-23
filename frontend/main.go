package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/avborup/ITU-DISYS2021-MiniProject3/service"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	bidderName string

	serverNodes = []string{"localhost:5000"}
	bidChannels = make([]chan int32, 0)

	currentBidMutex sync.Mutex
	currentBid      int32 = 0

	timeoutDuration = 5 * time.Second
)

func main() {
	setupBidder()
	dialAllNodes()

	stop := make(chan bool)
	<-stop
}

func makeBid(client service.ServiceClient, bid int32) {
	context, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	reply, err := client.MakeBid(context, &service.Bid{
		Amount: bid,
		Uuid:   bidderName,
	})

	if err != nil {
		log.Fatalf("Could not make bid, %s", err)
	}

	if reply.Status == service.Status_OK {
		updateBid(reply.Amount)
	}
}

func dialNode(addr string, channelIndex int) {

	client := service.NewServiceClient(getConnection(addr))

	updateBid(getResult(client).GetAmount())

	for {
		// Recipe for success:
		// 1. Get the current bid
		// 2. If the bid is higher than the current bid, update the current bid
		// 2b. Make a new bid that is higher than the updated bid
		// 2c. Send the new bid to the server
		// 3. If the bid is lower than the current bid, something went wrong and then just bid again
		// 4. If the bid is made by me, then I won't bid again

		// Check to see if there is a bid message in the channel and then make the bid

		log.Println("Going to check the bid channel for messages")
		select {
		case bid := <-bidChannels[channelIndex]:
			log.Printf("There were a bid in the channel. The bid is %d, the current bid is %d\n", bid, currentBid)
			makeBid(client, bid+1)
		default:
			log.Println("There were no bids to make")
		}

		// Get the latest auction result
		result := getResult(client)
		log.Printf("Got the latest result. Bid: %d, made by: %s", result.Amount, result.MadeBy)

		if result.Status == service.Status_AUCTION_OVER {
			log.Printf("The auction is over. The winner was %s\n", result.GetMadeBy())
			return
		}

		if result.GetMadeBy() == bidderName {
			log.Println("I was the highest bidder, so I won't bid again for now")
		} else {
			log.Printf("I'm going to bid!!!!. The current bid is %d, the latest result is %d\n", result.Amount, currentBid)
			updateBid(result.GetAmount())

			currentBidMutex.Lock()
			bidChannels[channelIndex] <- currentBid
			currentBidMutex.Unlock()
		}
	}
}

func dialAllNodes() {
	for i, addr := range serverNodes {
		// Create the bid channel for the coming goroutine
		bidChannels = append(bidChannels, make(chan int32, 10))

		go dialNode(addr, i)
	}
}

func updateBid(bid int32) {
	currentBidMutex.Lock()
	defer currentBidMutex.Unlock()

	if bid > currentBid {
		currentBid = bid
	}
}

func getResult(client service.ServiceClient) *service.Result {
	context, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	result, err := client.GetResult(context, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Could not get result, %s", err)
	}

	return result
}

func getConnection(addr string) *grpc.ClientConn {

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Could not connect to server %s", err)
	}
	return conn
}
