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

	timeoutDuration = 2 * time.Second

	rmDirectory = make(map[string]ReplicaManager)
)

type ReplicaManager struct {
	serviceClient service.ServiceClient
	address       string
}

func main() {
	log.Println("Starting to set up bidder")
	setupBidder()
	log.Println("Starting to dial all nodes")
	dialAllNodes()
	log.Println("Starting the bidding machine")
	bidMachine()

	stop := make(chan bool)
	<-stop
}

func getReplicaManagers() []ReplicaManager {

	replicaManagers := make([]ReplicaManager, 0)
	for _, nodeAddr := range serverNodes {
		if client, ok := rmDirectory[nodeAddr]; ok {
			replicaManagers = append(replicaManagers, client)
		} else {
			// the connection is lost, reconnect
			log.Printf("Trying to reconnect to node %s\n", nodeAddr)
			connection, err := getConnection(nodeAddr)

			if err == nil {
				client := service.NewServiceClient(connection)
				replicaManager := ReplicaManager{serviceClient: client, address: nodeAddr}
				replicaManagers = append(replicaManagers, replicaManager)
				rmDirectory[nodeAddr] = replicaManager

			} else {
				log.Printf("After attempted reconnection to %s, it failed again.\n", nodeAddr)
			}
		}
	}

	return replicaManagers
}

func makeBid(replicaManager ReplicaManager, bid int32) error {
	context, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	_, err := replicaManager.serviceClient.MakeBid(context, &service.Bid{
		Amount: bid,
		Uuid:   bidderName,
	})

	if err != nil {
		// The server disconnected: attempt to reconnect
		log.Printf("Could not make bid: %s\n", err)
		// remove map from the replica manager directory
		removeReplicaManager(replicaManager.address)
		return err
	}

	return nil
}

func bidMachine() {
	for {
		replicaManagers := getReplicaManagers()

		result := getCurrentHighestBid(&replicaManagers)

		if result.GetStatus() == service.Status_AUCTION_OVER {
			log.Printf("The auction is over. The winner was %s. The winning bid was: %d\n", result.GetMadeBy(), result.GetAmount())
			return
		}

		if result.GetMadeBy() != bidderName {
			newBid := result.GetAmount() + 1
			log.Printf("Client is going to make a new highest bid of %d\n", newBid)
			makeBidsOnAllReplicas(&replicaManagers, newBid)
		}
	}
}

func makeBidsOnAllReplicas(replicaManagers *[]ReplicaManager, bid int32) {

	wg := sync.WaitGroup{}
	for _, replicaManager := range *replicaManagers {

		wg.Add(1)
		go func(rm ReplicaManager) {

			makeBid(rm, bid)
			wg.Done()

		}(replicaManager)
	}
	wg.Wait()
}

func getCurrentHighestBid(replicaManagers *[]ReplicaManager) service.Result {

	var resultBuffer []service.Result
	resultLock := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, replicaManager := range *replicaManagers {

		// async boooi
		wg.Add(1)
		go func(rm ReplicaManager) {
			result, err := getResult(rm)

			if err != nil {
				wg.Done()
				return
			}

			resultLock.Lock()
			resultBuffer = append(resultBuffer, service.Result{MadeBy: result.GetMadeBy(),
				Amount: result.GetAmount(),
				Status: result.GetStatus()})
			resultLock.Unlock()

			wg.Done()
		}(replicaManager)
	}

	wg.Wait()

	bestResult := deepCopyResult(&resultBuffer[0])
	// find the highest result, such that we avoid propagating discrepancies
	for i, result := range resultBuffer {
		if i != 0 {
			if result.Amount > bestResult.Amount {
				bestResult = deepCopyResult(&result)
			}
		}
	}

	// returning a deep copy of an already deep copied result is to comply with the go formatter
	return deepCopyResult(&bestResult)
}

func deepCopyResult(result *service.Result) service.Result {
	return service.Result{MadeBy: result.GetMadeBy(),
		Amount: result.GetAmount(),
		Status: result.GetStatus(),
	}
}

func dialNode(addr string) (service.ServiceClient, error) {

	connection, err := getConnection(addr)
	if err != nil {
		// uh oh
		return nil, err
	}

	client := service.NewServiceClient(connection)
	return client, nil
}

func dialAllNodes() {
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

	rmLock := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, addr := range serverNodes {
		wg.Add(1)
		go func(address string) {
			serviceClient, _ := dialNode(address)

			rmLock.Lock()
			rmDirectory[address] = ReplicaManager{serviceClient: serviceClient, address: address}
			rmLock.Unlock()
			wg.Done()
		}(addr)
	}

	wg.Wait()
}

func getResult(replicaManager ReplicaManager) (*service.Result, error) {
	context, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	result, err := replicaManager.serviceClient.GetResult(context, &emptypb.Empty{})
	if err != nil {
		// the connection got lost - lets remove the client for now
		log.Printf("Could not get result: %s\n", err)
		removeReplicaManager(replicaManager.address)
	}

	return result, err
}

func removeReplicaManager(addr string) {
	delete(rmDirectory, addr)
}

func getConnection(addr string) (*grpc.ClientConn, error) {

	timeoutContext, cancelFunction := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancelFunction()

	conn, err := grpc.DialContext(timeoutContext, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("could not connect to server %s", err)
	}

	return conn, nil
}
