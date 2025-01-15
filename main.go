package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	pb "github.com/lshtar13/gRPC-chat/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct {
	pb.UnimplementedChatServer
}

func (s *server) BasicSend(_ context.Context, msg *pb.Msg) (*pb.Ack, error) {
	txt := msg.Msg
	sendTime, _ := time.Parse(time.RFC3339, msg.SendTime)
	fmt.Printf("%s : %s\n", sendTime.Format("2006-01-02 15:04:05"), txt)
	return &pb.Ack{}, nil
}

func (s *server) ContSend(stream pb.Chat_ContSendServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Ack{})
		} else if err != nil {
			return err
		}

		txt := in.Msg
		sendTime, _ := time.Parse(time.RFC3339, in.SendTime)
		fmt.Printf("%s : %s\n", sendTime.Format("2006-01-02 15:04:05"), txt)
	}
}

func startServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("fail to listen %v\n", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServer(s, &server{})
	log.Printf("Start Server at %d ...\n", port)
	s.Serve(lis)
}

func startClient(addr string, isBasic bool) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect with %s\n", addr)
	}
	defer conn.Close()

	c := pb.NewChatClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	location, _ := time.LoadLocation("Asia/Seoul")

	reader := bufio.NewScanner(os.Stdin)
	if isBasic {
		for {
			reader.Scan()
			msg := reader.Text()
			sendTime := time.Now().In(location).Format(time.RFC3339)

			if msg == "done" {
				break
			}

			_, err := c.BasicSend(ctx, &pb.Msg{Msg: msg, SendTime: sendTime})
			if err != nil {
				log.Fatalf("Failed to send %s to %s\n", msg, addr)
			}
		}
	} else {
		stream, err := c.ContSend(ctx)
		if err != nil {
			log.Fatalf("Failed to connect with %s", addr)
		}

		for {
			reader.Scan()
			msg := reader.Text()
			sendTime := time.Now().In(location).Format(time.RFC3339)

			if msg == "done" {
				_, err := stream.CloseAndRecv()
				if err != nil {
					log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
				}
				break
			}

			if err := stream.Send(&pb.Msg{Msg: msg, SendTime: sendTime}); err != nil {
				log.Fatalf("Failed to send %s to %s\n", msg, addr)
			}
		}
	}
}

func main() {
	// start server
	var port int
	fmt.Printf("Enter Port: ")
	fmt.Scan(&port)
	go startServer(port)

	time.Sleep(time.Second)

	// start client
	var addr string
	fmt.Printf("Enter Address : ")
	fmt.Scan(&addr)

	var isBasicStr string
	fmt.Printf("is Basic (true/false)?: ")
	fmt.Scan(&isBasicStr)
	isBasic, err := strconv.ParseBool(isBasicStr)
	if err != nil {
		log.Fatalf("choose true or false %s\n", err)
	}

	startClient(addr, isBasic)
}
