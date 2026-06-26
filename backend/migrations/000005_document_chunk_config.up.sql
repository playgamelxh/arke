ALTER TABLE documents
  ADD COLUMN chunk_size INT NOT NULL DEFAULT 500 AFTER chunk_strategy,
  ADD COLUMN chunk_overlap INT NOT NULL DEFAULT 50 AFTER chunk_size;
