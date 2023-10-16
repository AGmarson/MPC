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
	"sync"
	"time"

	exchange "github.com/AGmarson/MPC/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var hospitalId int32 = 5000
var prime int = 39916801
var receivedMessages int
var mutex sync.Mutex

func main() {
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
			log.Fatal("failed to create cert", err)
		}

		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithTransportCredentials(clientCert), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %s", err)
		}
		defer conn.Close()
		c := exchange.NewExchangeDataClient(conn)
		p.clients[port] = c
	}
	scanner := bufio.NewScanner(os.Stdin)
	if ownPort != hospitalId {
		fmt.Print("Enter a number 0 - 1.000.000 to secretly share it.\nNumber: ")
		for scanner.Scan() {
			secret, err := strconv.ParseInt(scanner.Text(), 10, 32)
			if err != nil || secret < 0 || secret > 1000000 {
				fmt.Print("Number must be between 0 and 1.000.000\nNumber: ")
			} else {
				p.ShareDataChunks(int(secret))
			}
		}
	} else {
		fmt.Print("Waiting for data from peers...\nWrite 'quit' to end me\n")
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
	mutex.Lock()
	chunk := req.SentData
	if receivedMessages == 2 && p.id != hospitalId { //if protocol followed correctly, the third message should be the result from hospital
		fmt.Printf("Received result: %v\nNumber: ", chunk)
		receivedMessages = 0
		mutex.Unlock()
		return &exchange.Reply{ReceivedData: chunk}, nil
	}
	receivedMessages++
	p.receivedChunks = append(p.receivedChunks, int(chunk))
	mutex.Unlock()
	if len(p.receivedChunks) == 3 {
		go func() {
			time.Sleep(time.Millisecond * 5) //give time to reply before doing extra stuff
			p.ProcessResult()
		}()
	}
	rep := &exchange.Reply{ReceivedData: chunk} //this is what I received
	return rep, nil
}

func (p *peer) ProcessResult() {
	result := (p.receivedChunks[0] + p.receivedChunks[1] + p.receivedChunks[2]) % prime
	if p.id == hospitalId {
		p.BroadCastResult(result)
	} else {
		p.SendDataTo(result, hospitalId)
	}
	p.receivedChunks = []int{}
}

func (p *peer) BroadCastResult(value int) {
	for id := range p.clients {
		if id == p.id {
			continue
		}
		p.SendDataTo(value, id)
	}
}

func (p *peer) ShareDataChunks(secret int) {
	chunks := ToChunks(secret)
	i := 0
	for id := range p.clients {
		if id == hospitalId || id == p.id {
			continue
		}
		p.SendDataTo(chunks[i], id)
		i++
	}
	fmt.Printf("Keeping last chunk %v for myself\n", chunks[i])
	p.receivedChunks = append(p.receivedChunks, chunks[i])
	if len(p.receivedChunks) == 3 {
		result := (p.receivedChunks[0] + p.receivedChunks[1] + p.receivedChunks[2]) % prime
		p.SendDataTo(result, hospitalId)
		p.receivedChunks = []int{}
	}
}

func (p *peer) SendDataTo(data int, id int32) {
	client := p.clients[id]
	request := &exchange.Request{SentData: int32(data)}
	fmt.Printf("Sending %v to %v... ", data, id)
	reply, err := client.Exchange(p.ctx, request)
	if err != nil {
		log.Printf("something went wrong: %v\n", err)
	}
	fmt.Printf("%v received: %v\n", id, reply.ReceivedData)
}

func ToChunks(secret int) []int {
	rand.Seed(time.Now().UnixNano())
	a := rand.Intn(prime - 1)
	b := rand.Intn(prime - 1)
	c := (secret - ((a + b) % prime)) % prime
	if c < 0 {
		c = prime + c //calculate the actual modulo (-1 % 5 = 4, not -1)
	}
	return []int{a, b, c}
}
