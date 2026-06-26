ALTER TABLE document_segments
  DROP INDEX idx_document_segments_vector,
  DROP COLUMN indexed_at,
  DROP COLUMN vector_id;

ALTER TABLE documents
  DROP FOREIGN KEY fk_documents_kb,
  DROP INDEX idx_documents_kb,
  DROP COLUMN knowledge_base_id;

DROP TABLE IF EXISTS knowledge_bases;
