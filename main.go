package main

import (
	"chat/proto"
	"context"
	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"
	"log"
	"net"
	"os"
	"sync"
)

var grpcLog glog.LoggerV2

func init() {
	file, err := os.OpenFile("logs.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	grpcLog = glog.NewLoggerV2(file, file, file)
}

type Connection struct {
	stream proto.Broadcast_CreateStreamServer
	id     string
	active bool
	error  chan error
}

type Server struct {
	Connection []*Connection
}

func (s *Server) CreateStream(pconn *proto.Connect, stream proto.Broadcast_CreateStreamServer) error {
	conn := &Connection{
		stream: stream,
		id:     pconn.User.Id,
		active: true,
		error:  make(chan error),
	}

	s.Connection = append(s.Connection, conn)

	return <-conn.error
}

func (s *Server) BroadcastMessage(ctx context.Context, msg *proto.Message) (*proto.Close, error) {
	wg := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.Connection {
		wg.Add(1)
		go func(msg *proto.Message, conn *Connection) {
			defer wg.Done()

			if conn.active {
				if err := conn.stream.Send(msg); err != nil {
					grpcLog.Errorf("Error with stream: %s - Error: %s", conn.stream, err.Error())
					conn.active = false
					conn.error <- err
				}
				grpcLog.Info("Sending message to: ", conn.stream)
			}

		}(msg, conn)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	<-done
	return &proto.Close{}, nil
}

func main() {
	var connections []*Connection

	server := &Server{connections}

	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", ":8001")
	if err != nil {
		log.Fatalf("error creating the server: %s", err)
	}

	grpcLog.Info("starting server on port: 8001")

	proto.RegisterBroadcastServer(grpcServer, server)
	grpcServer.Serve(lis)
}
