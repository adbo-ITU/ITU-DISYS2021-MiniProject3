package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/avborup/ITU-DISYS2021-MiniProject3/service"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	bidderName string

	serverNodes []string
	bidChannels = make([]chan int32, 0)

	currentBidMutex sync.Mutex
	currentBid      int32 = 0

	timeoutDuration = 5 * time.Second
)

func main() {
	log.Println("Starting to set up bidder")
	setupBidder()
	log.Println("Starting to dial all nodes")
	dialAllNodes()

	stop := make(chan bool)
	<-stop
}

func makeBid(client service.ServiceClient, bid int32) error {
	context, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	reply, err := client.MakeBid(context, &service.Bid{
		Amount: bid,
		Uuid:   bidderName,
	})

	if err != nil {
		// Ther server disconnected
		log.Printf("Could not make bid: %s\n", err)
		return err
	}

	if reply.Status == service.Status_OK {
		updateBid(reply.Amount)
	}

	return nil
}

func removeServerNode(index int) {
	newServerNodes := make([]string, 0)
	for i, addr := range serverNodes {
		if i != index {
			newServerNodes = append(newServerNodes, addr)
		}
	}
	serverNodes = newServerNodes
}

func dialNode(addr string, channelIndex int) {

	client := service.NewServiceClient(getConnection(addr))

	result, err := getResult(client)
	if err != nil {
		log.Println("Client could not get initial result from server")
		return
	}
	updateBid(result.GetAmount())

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
			err := makeBid(client, bid+1)

			if err != nil {
				removeServerNode(channelIndex)
				break
			}

		default:
			log.Println("There were no bids to make")
		}

		// Get the latest auction result
		result, err = getResult(client)

		if err != nil {
			removeServerNode(channelIndex)
			break
		}

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

	if len(serverNodes) == 0 {
		log.Fatalln("All servers are gone!")
	}
}

func dialAllNodes() {
	// Recipe for success
	// Read how many servers there are
	// use the prefix and then numbering to connect to all of the servers

	servers := os.Getenv("SERVERS")
	n, err := strconv.Atoi(servers)
	if err != nil {
		log.Fatalf("Could not convert number of servers variable to an int. Check its content: %s, err: %d", servers, err)
	}

	if n < 1 {
		log.Fatalf("There were less than 1 server configured to connect to. Shutting down program.")
	}

	// Create the addressses and add them to the server nodes slice
	for i := 0; i < n; i++ {
		addr := fmt.Sprintf("auctionserver%d:5000", i+1)
		serverNodes = append(serverNodes, addr)
	}

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

func getResult(client service.ServiceClient) (*service.Result, error) {
	context, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	result, err := client.GetResult(context, &emptypb.Empty{})
	if err != nil {

		// The server has crashed, so we should just kill the connection to the server
		log.Printf("Could not get result: %s\n", err)
	}

	return result, err
}

func getConnection(addr string) *grpc.ClientConn {

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Could not connect to server %s", err)
	}
	return conn
}
