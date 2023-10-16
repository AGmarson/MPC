package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	exchange "github.com/AGmarson/MPC/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var hospitalId int32 = 5000
var prime int = 39916801

func main() {
	log.SetFlags(log.Lshortfile)
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + hospitalId //clients are 5001 5002 5003

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:             ownPort,
		clients:        make(map[int32]exchange.ExchangeDataClient),
		receivedChunks: []int{},
		ctx:            ctx,
	}

	//set up server
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", ownPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverCert, err := credentials.NewServerTLSFromFile("certificate/server.crt", "certificate/priv.key")
	if err != nil {
		log.Fatalln("failed to create cert", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(serverCert))
	exchange.RegisterExchangeDataServer(grpcServer, p)

	// start the server
	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()

	//Dial the other peers
	for i := 0; i <= 3; i++ {
		port := hospitalId + int32(i)

		if port == ownPort {
			continue
		}

		//Set up client connections
		clientCert, err := credentials.NewClientTLSFromFile("certificate/server.crt", "")
		if err != nil {
			log.Fatalln("failed to create cert", err)
		}

		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithTransportCredentials(clientCert), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()
		c := exchange.NewExchangeDataClient(conn)
		p.clients[port] = c
		fmt.Printf("%v", p.clients)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if ownPort != hospitalId {
		fmt.Print("Enter a number between 0 and 1 000 000 to share it secretly with the other peers.\nNumber: ")
		for scanner.Scan() {
			secret, _ := strconv.ParseInt(scanner.Text(), 10, 32)
			p.ShareDataChunks(int(secret))
		}
	} else {
		fmt.Print("Waiting for data from peers...\nwrite 'quit' to end me\n")
		for scanner.Scan() {
			if scanner.Text() == "quit" {
				return
			}
		}
	}
}

type peer struct {
	exchange.UnimplementedExchangeDataServer
	id             int32
	clients        map[int32]exchange.ExchangeDataClient
	receivedChunks []int
	ctx            context.Context
}

func (p *peer) Exchange(ctx context.Context, req *exchange.Request) (*exchange.Reply, error) {
	fmt.Print("Here")
	chunk := req.SentData
	p.receivedChunks = append(p.receivedChunks, int(chunk))
	if len(p.receivedChunks) == 3 {
		result := (p.receivedChunks[0] + p.receivedChunks[1] + p.receivedChunks[2]) % prime
		if p.id == hospitalId {
			p.BroadCast(result)
		} else {
			p.SendToHospital(result)
		}
		p.receivedChunks = []int{}
	}
	rep := &exchange.Reply{ReceivedData: chunk} //this is what I received
	return rep, nil
}

func (p *peer) SendToHospital(value int) {
	hospital := p.clients[hospitalId]
	request := &exchange.Request{SentData: int32(value)}
	fmt.Printf("Sending %v to Hospital\n", request.SentData)
	reply, err := hospital.Exchange(p.ctx, request)
	if err != nil {
		log.Printf("something went wrong: %v\n", err)
	}
	fmt.Printf("Got reply from Hospital: I got %v\n", reply.ReceivedData)
}

func (p *peer) BroadCast(value int) {
	for id, client := range p.clients {
		if id == p.id {
			continue
		}
		request := &exchange.Request{SentData: int32(value)}
		fmt.Printf("Sending %v to %v\n", request.SentData, id)
		reply, err := client.Exchange(p.ctx, request)
		if err != nil {
			log.Printf("something went wrong: %v\n", err)
		}
		fmt.Printf("Got reply from id %v: I got %v\n", id, reply.ReceivedData)
	}
}

func (p *peer) ShareDataChunks(secret int) {
	chunks := ToChunks(secret)
	i := 0
	for id, client := range p.clients {
		if id == hospitalId || id == p.id {
			continue
		}
		request := &exchange.Request{SentData: int32(chunks[i])}
		i++
		fmt.Printf("Sending %v to %v\n", request.SentData, id)
		reply, err := client.Exchange(p.ctx, request)
		if err != nil {
			log.Printf("something went wrong: %v\n", err)
		}
		fmt.Printf("Got reply from id %v: I got %v\n", id, reply.ReceivedData)
	}
	fmt.Printf("Keeping %v for myself\n", chunks[i])
	p.receivedChunks = append(p.receivedChunks, chunks[i])
	if len(p.receivedChunks) == 3 {
		result := (p.receivedChunks[0] + p.receivedChunks[1] + p.receivedChunks[2]) % prime
		p.SendToHospital(result)
		p.receivedChunks = []int{}
	}
}

func ToChunks(secret int) []int {
	rand.Seed(time.Now().UnixNano())
	a := rand.Intn(prime - 1)
	b := rand.Intn(prime - 1)
	c := (secret - ((a + b) % prime)) % prime
	if c < 0 {
		c = prime + c //calculate the actual modulo
	}
	return []int{a, b, c}
}
