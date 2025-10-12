-- Modify "documents" table
ALTER TABLE "documents" DROP COLUMN "storage_path", ADD COLUMN "content" text NOT NULL, ADD COLUMN "content_hash" character varying NULL;
-- Create index "document_content_hash" to table: "documents"
CREATE INDEX "document_content_hash" ON "documents" ("content_hash");
