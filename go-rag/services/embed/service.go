package embed

import (
	"context"
	"go-rag/ent/ent"
	"go-rag/services/proto"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Service handles the document processing pipeline.
type Service struct {
	Client          *ent.Client
	InferenceClient proto.InferencerClient
}

type embeddingJob struct {
	Index int   // To maintain order
	Chunk Chunk // From your chunker.go
}

type embeddingResult struct {
	Index  int // To match the original chunk
	Vector []float32
	Err    error
}

// ProcessDocument is the main entry point for processing a single document asynchronously.
func (s *Service) ProcessDocument(ctx context.Context, documentID int) {
	log := logrus.WithField("document_id", documentID)
	log.Info("starting processing for document")

	// 1. Fetch document and set status to "processing".
	doc, err := s.Client.Document.UpdateOneID(documentID).SetStatus("processing").Save(ctx)
	if err != nil {
		log.WithError(err).Error("failed to fetch document or update status")
		return
	}

	// 2. Choose chunking strategy based on file type.
	var chunks []Chunk
	if strings.HasSuffix(doc.Name, ".md") {
		chunks = ChunkMarkdown(doc.Content) // Using your new precise chunker
	} else {
		chunks = chunkCodeFile(doc.Content) // Using the placeholder for code
	}
	log.WithField("chunk_count", len(chunks)).Info("document chunked")

	// 3. Embed all chunks concurrently using a worker pool.
	vectors, err := s.embedChunks(ctx, chunks)
	if err != nil {
		log.WithError(err).Error("failed to embed chunks")
		s.Client.Document.UpdateOneID(doc.ID).SetStatus("failed").Exec(ctx)
		return
	}
	log.Info("all chunks embedded successfully")

	// 4. Save chunks and embeddings to the database in a single transaction.
	tx, err := s.Client.Tx(ctx)
	if err != nil {
		log.WithError(err).Error("failed to start transaction")
		s.Client.Document.UpdateOneID(doc.ID).SetStatus("failed").Exec(ctx)
		return
	}

	for i, chunkData := range chunks {
		c, err := tx.Chunk.Create().SetIndex(i).SetContent(chunkData.Content).SetDocument(doc).Save(ctx)
		if err != nil {
			log.WithError(err).Error("failed to save chunk, rolling back transaction")
			tx.Rollback()
			return
		}
		_, err = tx.Embedding.Create().SetVector(vectors[i]).SetChunk(c).Save(ctx)
		if err != nil {
			log.WithError(err).Error("failed to save embedding, rolling back transaction")
			tx.Rollback()
			return
		}
	}

	// 5. Commit transaction and update document status to "completed".
	if err := tx.Commit(); err != nil {
		log.WithError(err).Error("failed to commit transaction")
		s.Client.Document.UpdateOneID(doc.ID).SetStatus("failed").Exec(ctx)
		return
	}

	s.Client.Document.UpdateOneID(doc.ID).SetStatus("completed").Save(ctx)
	log.Info("document processing completed successfully")
}

// embedChunks manages a pool of goroutines to embed chunks in parallel.
func (s *Service) embedChunks(ctx context.Context, chunks []Chunk) ([][]float32, error) {
	numJobs := len(chunks)
	jobs := make(chan embeddingJob, numJobs)
	results := make(chan embeddingResult, numJobs)
	numWorkers := 10 // This is the number of concurrent goroutines. Tune as needed.
	var wg sync.WaitGroup

	// Start the worker goroutines.
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go s.embeddingWorker(ctx, &wg, jobs, results)
	}

	// Send all the chunks to the jobs channel.
	for i, chunk := range chunks {
		jobs <- embeddingJob{Index: i, Chunk: chunk}
	}
	close(jobs)

	// Wait for all the workers to finish.
	wg.Wait()
	close(results)

	// Collect the results, ensuring they are in the correct order.
	finalVectors := make([][]float32, numJobs)
	for res := range results {
		if res.Err != nil {
			return nil, res.Err // On first error, fail the whole batch.
		}
		finalVectors[res.Index] = res.Vector
	}
	return finalVectors, nil
}

// embeddingWorker is a single goroutine that pulls jobs, calls the gRPC service, and sends results.
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
