package embed

import (
	"context"
	"fmt"
	"go-rag/ent/ent"
	"go-rag/ent/ent/chunk"
	"go-rag/ent/ent/document"
	"go-rag/services/proto"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sirupsen/logrus"
)

const CollectionName = "go-rag-chunks"

// Service handles the document processing pipeline.
type Service struct {
	Client             *ent.Client
	InferenceClient    proto.InferencerClient
	QdrantPointsClient qdrant.PointsClient
}

type embeddingJob struct {
	Index int
	Chunk Chunk
}

type embeddingResult struct {
	Index  int
	Vector []float32
	Err    error
}

// ProcessDocument handles the intelligent chunking and embedding of a document.
func (s *Service) ProcessDocument(ctx context.Context, documentID int) {
	log := logrus.WithField("document_id", documentID)
	log.Info("starting smart processing for document")

	// 1. Fetch document, its project, owner, and existing chunks.
	doc, err := s.Client.Document.Query().
		Where(document.ID(documentID)).
		WithProject(func(q *ent.ProjectQuery) {
			q.WithOwner() // Eager-load the owner of the project
		}).
		WithChunks().
		Only(ctx)
	if err != nil {
		log.WithError(err).Error("failed to fetch document with relations")
		s.Client.Document.UpdateOneID(documentID).SetStatus("failed").Exec(context.Background())
		return
	}

	ownerID := doc.Edges.Project.Edges.Owner.ID

	// Create a map of existing chunk hashes for quick lookups.
	existingChunks := make(map[string]*ent.Chunk)
	for _, c := range doc.Edges.Chunks {
		existingChunks[c.ContentHash] = c
	}

	// 2. Generate new chunks from the document's content.
	var newChunks []Chunk
	if strings.HasSuffix(doc.Name, ".md") {
		newChunks = ChunkMarkdown(doc.Content)
	} else {
		newChunks = chunkCodeFile(doc.Content)
	}
	log.WithFields(logrus.Fields{
		"new_chunk_count":      len(newChunks),
		"existing_chunk_count": len(existingChunks),
	}).Info("document chunked and existing chunks loaded")

	// 3. Determine which chunks are new, modified, or deleted.
	var chunksToEmbed []Chunk
	chunksToDelete := make(map[string]*ent.Chunk)
	for k, v := range existingChunks {
		chunksToDelete[k] = v // Assume all old chunks will be deleted initially
	}

	for _, newChunk := range newChunks {
		if _, exists := existingChunks[newChunk.ContentHash]; exists {
			// This chunk is unchanged. Remove it from the deletion list.
			delete(chunksToDelete, newChunk.ContentHash)
		} else {
			// This is a new or modified chunk that needs embedding.
			chunksToEmbed = append(chunksToEmbed, newChunk)
		}
	}

	log.WithFields(logrus.Fields{
		"to_embed":  len(chunksToEmbed),
		"to_delete": len(chunksToDelete),
	}).Info("calculated chunk diff")

	// 4. Process the diff.
	if len(chunksToEmbed) > 0 || len(chunksToDelete) > 0 {
		var vectors [][]float32
		if len(chunksToEmbed) > 0 {
			var err error
			vectors, err = s.embedChunks(ctx, chunksToEmbed)
			if err != nil {
				log.WithError(err).Error("failed to embed new/modified chunks")
				s.Client.Document.UpdateOneID(doc.ID).SetStatus("failed").Exec(ctx)
				return
			}
			log.Info("new chunks embedded successfully")
		}

		// 5. Save everything to the databases (Postgres + Qdrant).
		if err := s.syncDatabase(ctx, doc, ownerID, chunksToEmbed, vectors, chunksToDelete); err != nil {
			log.WithError(err).Error("failed to sync databases")
			s.Client.Document.UpdateOneID(doc.ID).SetStatus("failed").Exec(ctx)
			return
		}
	} else {
		log.Info("no changes detected in document content")
	}

	// 6. Finalize document status.
	s.Client.Document.UpdateOneID(doc.ID).SetStatus("completed").SaveX(ctx)
	log.Info("document smart processing completed successfully")
}

func (s *Service) DeleteDocumentVectors(ctx context.Context, documentID int) error {
	log := logrus.WithField("document_id", documentID)
	log.Info("deleting all vectors for document from Qdrant")

	// Find all chunk IDs for the given document.
	chunkIDs, err := s.Client.Document.Query().
		Where(document.ID(documentID)).
		QueryChunks().
		IDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to query chunk IDs for document: %w", err)
	}

	if len(chunkIDs) == 0 {
		log.Warn("no chunks found for document, nothing to delete from Qdrant")
		return nil
	}

	// Convert chunk IDs to Qdrant PointId format.
	var pointsToDelete []*qdrant.PointId
	for _, id := range chunkIDs {
		pointsToDelete = append(pointsToDelete, &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Num{Num: uint64(id)},
		})
	}

	// Create the delete request.
	wait := true
	pointsSelector := &qdrant.PointsSelector{
		PointsSelectorOneOf: &qdrant.PointsSelector_Points{
			Points: &qdrant.PointsIdsList{
				Ids: pointsToDelete,
			},
		},
	}

	// Execute the delete operation.
	_, err = s.QdrantPointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: CollectionName,
		Points:         pointsSelector,
		Wait:           &wait,
	})

	if err != nil {
		return fmt.Errorf("failed to delete points from qdrant: %w", err)
	}

	log.WithField("count", len(pointsToDelete)).Info("successfully deleted document vectors from Qdrant")
	return nil
}

// syncDatabase handles the transactional update to Postgres and the corresponding upsert/delete in Qdrant.
func (s *Service) syncDatabase(ctx context.Context, doc *ent.Document, ownerID uuid.UUID, newChunks []Chunk, newVectors [][]float32, chunksToDelete map[string]*ent.Chunk) error {
	// --- Delete old points from Qdrant ---
	if len(chunksToDelete) > 0 {
		var pointsToDelete []*qdrant.PointId
		for _, c := range chunksToDelete {
			pointsToDelete = append(pointsToDelete, &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Num{Num: uint64(c.ID)},
			})
		}

		wait := true
		pointsSelector := &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointsToDelete,
				},
			},
		}

		_, err := s.QdrantPointsClient.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: CollectionName,
			Points:         pointsSelector,
			Wait:           &wait,
		})
		if err != nil {
			return fmt.Errorf("failed to delete points from qdrant: %w", err)
		}
		logrus.WithField("count", len(pointsToDelete)).Info("deleted old points from qdrant")
	}

	// --- Start Postgres Transaction ---
	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Delete old chunks from Postgres
	var idsToDelete []int
	for _, c := range chunksToDelete {
		idsToDelete = append(idsToDelete, c.ID)
	}
	if len(idsToDelete) > 0 {
		if _, err := tx.Chunk.Delete().Where(chunk.IDIn(idsToDelete...)).Exec(ctx); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete old chunks from postgres: %w", err)
		}
		logrus.WithField("count", len(idsToDelete)).Info("deleted old chunks from postgres")
	}

	// Create new chunks in Postgres and prepare points for Qdrant
	var pointsToUpsert []*qdrant.PointStruct
	for i, chunkData := range newChunks {
		c, err := tx.Chunk.Create().
			SetIndex(i).
			SetContent(chunkData.Content).
			SetContentHash(chunkData.ContentHash).
			SetDocument(doc).
			Save(ctx)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to save new chunk: %w", err)
		}

		// Prepare the point for Qdrant with the rich payload
		pointsToUpsert = append(pointsToUpsert, &qdrant.PointStruct{
			Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Num{Num: uint64(c.ID)}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: newVectors[i]}}},
			Payload: map[string]*qdrant.Value{
				"user_id":     {Kind: &qdrant.Value_StringValue{StringValue: ownerID.String()}},
				"project_id":  {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(doc.Edges.Project.ID)}},
				"document_id": {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(doc.ID)}},
				"chunk_id":    {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(c.ID)}},
			},
		})
	}

	// Upsert new points to Qdrant
	if len(pointsToUpsert) > 0 {
		wait := true
		_, err := s.QdrantPointsClient.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: CollectionName,
			Points:         pointsToUpsert,
			Wait:           &wait,
		})
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert new points to qdrant: %w", err)
		}
		logrus.WithField("count", len(pointsToUpsert)).Info("upserted new points to qdrant")
	}

	return tx.Commit()
}

// embedChunks manages a pool of goroutines to embed chunks in parallel.
func (s *Service) embedChunks(ctx context.Context, chunks []Chunk) ([][]float32, error) {
	numJobs := len(chunks)
	if numJobs == 0 {
		return nil, nil
	}
	jobs := make(chan embeddingJob, numJobs)
	results := make(chan embeddingResult, numJobs)
	numWorkers := 10 // Concurrent goroutines
	var wg sync.WaitGroup

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go s.embeddingWorker(ctx, &wg, jobs, results)
	}

	for i, chunk := range chunks {
		jobs <- embeddingJob{Index: i, Chunk: chunk}
	}
	close(jobs)

	wg.Wait()
	close(results)

	finalVectors := make([][]float32, numJobs)
	for res := range results {
		if res.Err != nil {
			return nil, res.Err
		}
		finalVectors[res.Index] = res.Vector
	}
	return finalVectors, nil
}

// embeddingWorker is a single goroutine that calls the gRPC service.
func (s *Service) embeddingWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan embeddingJob, results chan<- embeddingResult) {
	defer wg.Done()
	for job := range jobs {
		req := &proto.EmbeddingRequest{Text: job.Chunk.Content}
		res, err := s.InferenceClient.GetEmbedding(ctx, req)
		var vector []float32
		if res != nil {
			vector = res.Embedding
		}
		results <- embeddingResult{
			Index:  job.Index,
			Vector: vector,
			Err:    err,
		}
	}
}
