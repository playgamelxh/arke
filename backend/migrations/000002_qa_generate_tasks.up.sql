CREATE TABLE IF NOT EXISTS qa_generate_tasks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  document_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  progress INT NOT NULL DEFAULT 0,
  message VARCHAR(255) NOT NULL DEFAULT '',
  task_count INT NOT NULL DEFAULT 10,
  difficulty VARCHAR(32) NOT NULL DEFAULT 'normal',
  result_json LONGTEXT NULL,
  error_message TEXT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  INDEX idx_qa_generate_tasks_document_id (document_id),
  INDEX idx_qa_generate_tasks_status (status),
  CONSTRAINT fk_qa_generate_tasks_document FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
