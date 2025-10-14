package qdrant

import (
	"context"
	"fmt"
	"go-rag/services/proto"
	"os"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const VectorSize = 768

// NewClient establishes a gRPC connection to Qdrant and returns the clients.
func NewClient(ctx context.Context) (qdrant.PointsClient, qdrant.CollectionsClient, *grpc.ClientConn, error) {
	host := os.Getenv("QDRANT_SERVICE_HOST")
	port := os.Getenv("QDRANT_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, nil, nil, fmt.Errorf("QDRANT_HOST or QDRANT_GRPC_PORT is not set")
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	logrus.WithField("address", addr).Info("connecting to Qdrant gRPC service")

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.WithError(err).Error("failed to connect to Qdrant")
		return nil, nil, nil, fmt.Errorf("did not connect: %w", err)
	}

	pointsClient := qdrant.NewPointsClient(conn)
	collectionsClient := qdrant.NewCollectionsClient(conn)

	// Simple health check
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err = collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		logrus.WithError(err).Error("qdrant health check failed")
		conn.Close()
		return nil, nil, nil, fmt.Errorf("qdrant health check failed: %w", err)
	}

	logrus.Info("successfully connected to Qdrant")
	return pointsClient, collectionsClient, conn, nil
}

// EnsureCollectionExists checks if a collection exists and creates it with payload indexes if it doesn't.
func EnsureCollectionExists(ctx context.Context, collectionsClient qdrant.CollectionsClient, pointsClient qdrant.PointsClient, collectionName string) error {
	log := logrus.WithField("collection_name", collectionName)

	_, err := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})

	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			log.Info("collection not found, creating it now...")
			wait := true

			// Step 1: Create the collection itself.
			_, err := collectionsClient.Create(ctx, &qdrant.CreateCollection{
				CollectionName: collectionName,
				VectorsConfig: &qdrant.VectorsConfig{
					Config: &qdrant.VectorsConfig_Params{
						Params: &qdrant.VectorParams{
							Size:     VectorSize,
							Distance: qdrant.Distance_Cosine,
						},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("could not create collection: %w", err)
			}
			log.Info("collection created successfully, now creating payload indexes...")

			// Step 2: Create a payload index for each field using the correct enum values.
			_, err = pointsClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
				CollectionName: collectionName,
				FieldName:      "user_id",
				FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
				Wait:           &wait,
			})
			if err != nil {
				return fmt.Errorf("could not create 'user_id' payload index: %w", err)
			}
			_, err = pointsClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
				CollectionName: collectionName,
				FieldName:      "project_id",
				FieldType:      qdrant.FieldType_FieldTypeInteger.Enum(),
				Wait:           &wait,
			})
			if err != nil {
				return fmt.Errorf("could not create 'project_id' payload index: %w", err)
			}
			_, err = pointsClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
				CollectionName: collectionName,
				FieldName:      "document_id",
				FieldType:      qdrant.FieldType_FieldTypeInteger.Enum(),
				Wait:           &wait,
			})
			if err != nil {
				return fmt.Errorf("could not create 'document_id' payload index: %w", err)
			}

			_, err = pointsClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
				CollectionName: collectionName,
				FieldName:      "chunk_id",
				FieldType:      qdrant.FieldType_FieldTypeInteger.Enum(),
				Wait:           &wait,
			})
			if err != nil {
				return fmt.Errorf("could not create 'document_id' payload index: %w", err)
			}

			log.Info("all payload indexes created successfully")
			return nil
		}
		return fmt.Errorf("could not get collection info: %w", err)
	}

	log.Info("collection already exists")
	return nil
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
