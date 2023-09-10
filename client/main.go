package main

import (
	"bufio"
	"chat/proto"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"sync"
	"time"
)

var client proto.BroadcastClient
var wg *sync.WaitGroup

func init() {
	wg = &sync.WaitGroup{}
}

func connect(user *proto.User) error {
	var streamErr error

	stream, err := client.CreateStream(context.Background(), &proto.Connect{
		User:   user,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	wg.Add(1)
	go func(str proto.Broadcast_CreateStreamClient) {
		defer wg.Done()
		for {
			msg, err := str.Recv()
			if err != nil {
				streamErr = fmt.Errorf("error : %s", err)
				break
			}
			fmt.Printf("%d : %s \n", msg.Id, msg.Message)
		}
	}(stream)

	return streamErr
}

func main() {
	timestamp := time.Now()
	done := make(chan int)

	name := flag.String("N", "Anon", "the Name")
	flag.Parse()

	id := sha256.Sum256([]byte(timestamp.String() + *name))

	conn, err := grpc.Dial("localhost:8001", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("error : %s", err)
	}

	client = proto.NewBroadcastClient(conn)
	user := &proto.User{
		Id:   hex.EncodeToString(id[:]),
		Name: *name,
	}

	connect(user)

	wg.Add(1)
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := &proto.Message{
				Id:        user.Id,
				Message:   scanner.Text(),
				Timestamp: timestamp.String(),
			}

			_, err := client.BroadcastMessage(context.Background(), msg)
			if err != nil {
				fmt.Printf("error : %s", err)
				break
			}
		}
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	<-done
}
