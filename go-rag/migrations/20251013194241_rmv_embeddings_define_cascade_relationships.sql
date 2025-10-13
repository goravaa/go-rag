-- Modify "documents" table
ALTER TABLE "documents" DROP CONSTRAINT "documents_projects_documents", ADD CONSTRAINT "documents_projects_documents" FOREIGN KEY ("project_documents") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "chunks" table
ALTER TABLE "chunks" DROP CONSTRAINT "chunks_documents_chunks", ADD COLUMN "content_hash" character varying NULL, ADD CONSTRAINT "chunks_documents_chunks" FOREIGN KEY ("document_chunks") REFERENCES "documents" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "chunk_content_hash" to table: "chunks"
CREATE INDEX "chunk_content_hash" ON "chunks" ("content_hash");
-- Modify "query_results" table
ALTER TABLE "query_results" DROP COLUMN "document_query_results";
-- Create "chunk_query_results" table
CREATE TABLE "chunk_query_results" (
  "chunk_id" bigint NOT NULL,
  "query_result_id" bigint NOT NULL,
  PRIMARY KEY ("chunk_id", "query_result_id"),
  CONSTRAINT "chunk_query_results_chunk_id" FOREIGN KEY ("chunk_id") REFERENCES "chunks" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "chunk_query_results_query_result_id" FOREIGN KEY ("query_result_id") REFERENCES "query_results" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Modify "user_prompts" table
ALTER TABLE "user_prompts" DROP CONSTRAINT "user_prompts_projects_queries", ADD CONSTRAINT "user_prompts_projects_queries" FOREIGN KEY ("project_queries") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Drop "embeddings" table
DROP TABLE "embeddings";
