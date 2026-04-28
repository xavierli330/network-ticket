CREATE TABLE alert_sources (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    type VARCHAR(64) NOT NULL COMMENT 'zabbix | prometheus | generic',
    config JSON COMMENT '连接配置(认证方式等)',
    parser_config JSON COMMENT '字段映射配置(JSONPath)',
    webhook_secret VARCHAR(255) COMMENT 'webhook验签密钥',
    poll_endpoint VARCHAR(512) COMMENT '轮询地址',
    poll_interval INT DEFAULT 0 COMMENT '轮询间隔(秒), 0=不轮询',
    dedup_fields JSON COMMENT '去重指纹字段配置',
    dedup_window_sec INT DEFAULT 600 COMMENT '去重窗口(秒), 默认10分钟',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
