package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"testing"
	"time"

	pb "github.com/sdslabs/katanabroadcast-service/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestBroadcast(t *testing.T) {
	conn, err := grpc.Dial("localhost:3003", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln("Error connecting to broadcast service: ", err)
	}
	file, err := os.Open("run.zip")
	if err != nil {
		log.Fatal("Error reading file")
	}
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	fileUploadClient := pb.NewFileUploadServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := fileUploadClient.UploadFile(ctx)
	if err != nil {
		log.Fatal("Cannot send file: ", err)
	}

	req := &pb.UploadFileRequest{
		Data: &pb.UploadFileRequest_FileInfo{
			FileInfo: &pb.FileInfo{
				FileName: "run.zip",
				ChalName: "run",
			},
		},
	}

	err = stream.Send(req)
	if err != nil {
		log.Fatal("Cannot send challenge file: ", err, stream.RecvMsg(nil))
	}

	req = &pb.UploadFileRequest{
		Data: &pb.UploadFileRequest_Addresses{
			Addresses: &pb.PodAddresses{
				Address: []string{"localhost:5050"},
			},
		},
	}
	err = stream.Send(req)
	if err != nil {
		log.Fatalln("Error sending addresses: ", err)
	}
	fileSize := 0
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Cannot read file: ", err)
		}

		req := &pb.UploadFileRequest{
			Data: &pb.UploadFileRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}
		fileSize += n
		err = stream.Send(req)

		if err != nil {
			log.Fatal("Cannot send file chunk to server: ", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("Cannot receive response: ", err)
	}
	gotFileSize := res.GetSize()
	if gotFileSize != uint64(fileSize) {
		t.Error("Error")
	}
}
