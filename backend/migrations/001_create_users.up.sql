CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'operator' COMMENT 'admin | operator',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO users (username, password, role) VALUES
('admin', '$2a$10$wVfbWEnRXOidIOaQ2s6kGeTQrNxQucW.WhH/.rW1K7LsmP1TFIFTm', 'admin');
