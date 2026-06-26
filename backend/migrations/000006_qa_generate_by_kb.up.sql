ALTER TABLE qa_generate_tasks
  ADD COLUMN knowledge_base_id BIGINT UNSIGNED NULL AFTER id,
  ADD INDEX idx_qa_generate_tasks_kb (knowledge_base_id),
  ADD CONSTRAINT fk_qa_generate_tasks_kb FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(id) ON DELETE CASCADE;
