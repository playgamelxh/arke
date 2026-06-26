ALTER TABLE qa_generate_tasks
  DROP FOREIGN KEY fk_qa_generate_tasks_kb,
  DROP INDEX idx_qa_generate_tasks_kb,
  DROP COLUMN knowledge_base_id;
