package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"time"

	pb "github.com/sdslabs/katanabroadcast-service/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct {
	pb.UnimplementedFileUploadServiceServer
}

func (s *server) UploadFile(stream pb.FileUploadService_UploadFileServer) error {
	req, err := stream.Recv()
	if err != nil {
		log.Println("Error: ", err)
		return err
	}
	fileName := req.GetFileInfo().GetFileName()
	chalName := req.GetFileInfo().GetChalName()
	log.Printf("Recieving file %s of challenge %s", fileName, chalName)
	req, err = stream.Recv()
	if err != nil {
		log.Println("Error: ", err)
		return err
	}
	addresses := req.GetAddresses().GetAddress()
	fileData := bytes.Buffer{}
	fileSize := 0
	log.Print("Waiting to recive data")
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			log.Print("Finished Recieving")
			break
		}
		if err != nil {
			log.Println("Error: ", err)
			return err
		}
		chunk := req.GetChunkData()
		fileSize += len(chunk)
		_, err = fileData.Write(chunk)
		if err != nil {
			log.Println("Error: ", err)
			return err
		}
	}
	res := &pb.UploadFileResponse{
		Size: uint64(fileSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		log.Println("Error: ", err)
		return err
	}
	for _, address := range addresses {
		go SendFile(fileData, fileName, address)
	}
	return nil
}

func SendFile(fileData bytes.Buffer, filename, uri string) error {
	conn, err := grpc.Dial(uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln("Error connecting to teamvm ", err)
	}
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
				FileName: filename,
				ChalName: filename,
			},
		},
	}

	err = stream.Send(req)
	if err != nil {
		log.Fatal("Cannot send challenge file: ", err, stream.RecvMsg(nil))
	}
	fileSize := 0
	for {
		n, err := fileData.Read(buffer)
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
		log.Printf("Error sending file some bits maybe lost expected %d bytes, got %d bytes", fileSize, gotFileSize)
	}
	return nil
}

func setupServer() error {
	lis, err := net.Listen("tcp", ":3003")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterFileUploadServiceServer(grpcServer, &server{})
	grpcServer.Serve(lis)
	return nil
}

func main() {
	if err := setupServer(); err != nil {
		log.Fatalln(err)
	}
}
