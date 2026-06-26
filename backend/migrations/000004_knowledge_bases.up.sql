CREATE TABLE IF NOT EXISTS knowledge_bases (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  description TEXT NULL,
  embedding_model VARCHAR(128) NOT NULL DEFAULT 'text-embedding-v3',
  embedding_dim INT NOT NULL DEFAULT 1024,
  index_type VARCHAR(32) NOT NULL DEFAULT 'HNSW',
  index_params JSON NULL,
  chunk_strategy VARCHAR(32) NOT NULL DEFAULT 'paragraph',
  chunk_size INT NOT NULL DEFAULT 500,
  chunk_overlap INT NOT NULL DEFAULT 50,
  milvus_collection VARCHAR(128) NOT NULL,
  doc_count INT NOT NULL DEFAULT 0,
  vector_count BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_knowledge_bases_name (name),
  UNIQUE KEY uk_knowledge_bases_collection (milvus_collection)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE documents
  ADD COLUMN knowledge_base_id BIGINT UNSIGNED NULL AFTER id,
  ADD COLUMN chunk_strategy VARCHAR(32) NOT NULL DEFAULT 'paragraph' AFTER status,
  ADD INDEX idx_documents_kb (knowledge_base_id),
  ADD CONSTRAINT fk_documents_kb FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(id) ON DELETE CASCADE;

ALTER TABLE document_segments
  ADD COLUMN vector_id VARCHAR(64) NULL AFTER document_id,
  ADD COLUMN indexed_at DATETIME(3) NULL AFTER created_at,
  ADD INDEX idx_document_segments_vector (vector_id);
