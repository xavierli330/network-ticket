CREATE TABLE workflow_states (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id BIGINT UNSIGNED NOT NULL,
    node_name VARCHAR(64) NOT NULL COMMENT 'alert_received | parsed | pushed | awaiting_auth | authorized | executing | completed',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending | active | done | failed | skipped | timeout',
    operator VARCHAR(64) COMMENT 'system | client:xxx',
    input_data JSON,
    output_data JSON,
    error_message TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ticket_id (ticket_id),
    KEY idx_ticket_node (ticket_id, node_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
