ALTER TABLE tickets DROP FOREIGN KEY fk_tickets_ticket_type;
ALTER TABLE tickets DROP COLUMN ticket_type_id;
ALTER TABLE tickets MODIFY alert_source_id BIGINT UNSIGNED NOT NULL;
DROP TABLE ticket_types;
