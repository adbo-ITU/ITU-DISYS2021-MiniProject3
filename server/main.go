package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/avborup/ITU-DISYS2021-MiniProject3/service"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	auctionTime = 60 * time.Second
)

var (
	highestBid = service.Result{
		MadeBy: "server, lol make a bid man",
		Amount: 0,
		Status: service.Status_OK,
	}
	bidLocker sync.Mutex
)

func main() {
	go func() {
		time.Sleep(auctionTime)
		endAuction()
	}()

	StartServer()
}

type Server struct {
	service.UnimplementedServiceServer
}

func StartServer() {
	port := os.Getenv("PORT")
	address := fmt.Sprintf("0.0.0.0:%v", port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error while attempting to listen on port %v: %v", port, err)
	}

	log.Printf("Started server on %s", address)
	grpcServer := grpc.NewServer()

	server := Server{}
	service.RegisterServiceServer(grpcServer, &server)

	// Beware, Serve is blocking
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Server failed to serve: %v", err)
	}
}

func (s *Server) MakeBid(context context.Context, bid *service.Bid) (*service.Result, error) {
	bidLocker.Lock()
	defer bidLocker.Unlock()

	if highestBid.Status == service.Status_AUCTION_OVER {
		log.Printf("MakeBid: The bid %v by %v was illegal, auction is closed!\n", bid.Amount, bid.Uuid)
		return &service.Result{Status: service.Status_AUCTION_OVER}, nil
	}

	if bid.Amount <= highestBid.Amount {
		log.Printf("MakeBid: The bid %v by %v was too small, beaten by a bid of %v from %v!\n", bid.Amount, bid.Uuid, highestBid.Amount, highestBid.MadeBy)
		return &service.Result{Status: service.Status_TOO_LOW}, nil
	}

	log.Printf("MakeBid: A new highest bid of %v from %v has beaten the old bid of %v from %v!\n", bid.Amount, bid.Uuid, highestBid.Amount, highestBid.MadeBy)
	highestBid = service.Result{
		Amount: bid.Amount,
		MadeBy: bid.Uuid,
		Status: service.Status_OK,
	}
	return &highestBid, nil
}

func (s *Server) GetResult(context context.Context, _ *emptypb.Empty) (*service.Result, error) {
	bidLocker.Lock()
	defer bidLocker.Unlock()
	log.Printf("GetResult: highest bid is %v by %v\n", highestBid.Amount, highestBid.MadeBy)
	return &highestBid, nil
}

func endAuction() {
	bidLocker.Lock()
	defer bidLocker.Unlock()
	highestBid.Status = service.Status_AUCTION_OVER
	log.Printf("Auction ended - the winning bid was %v by %v\n!", highestBid.Amount, highestBid.MadeBy)
}
