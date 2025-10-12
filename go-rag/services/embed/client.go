package embed

import (
	"context"
	"fmt"
	"os"

	"go-rag/services/proto"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient creates and returns a new gRPC client for the inference service.
func NewClient() (proto.InferencerClient, *grpc.ClientConn, error) {
	host := os.Getenv("EMBEDDING_SERVICE_HOST")
	port := os.Getenv("EMBEDDING_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, nil, fmt.Errorf("EMBEDDING_SERVICE_HOST or EMBEDDING_SERVICE_PORT is not set")
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	logrus.WithField("address", addr).Info("connecting to embedding service")

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.WithError(err).Error("failed to connect to embedding service")
		return nil, nil, fmt.Errorf("did not connect: %w", err)
	}

	client := proto.NewInferencerClient(conn)
	logrus.Info("successfully connected to embedding service")
	return client, conn, nil
}

// TestEmbeddingCall performs a simple test RPC to verify the client works.
func TestEmbeddingCall(client proto.InferencerClient, text string) error {
	ctx := context.Background()
	resp, err := client.GetEmbedding(ctx, &proto.EmbeddingRequest{Text: text})
	if err != nil {
		logrus.WithError(err).Error("RPC call failed")
		return err
	}

	logrus.WithField("embedding_length", len(resp.Embedding)).
		Infof("Received embedding for text: %q", text)
	return nil
}
