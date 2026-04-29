-- 创建工单类型表
CREATE TABLE ticket_types (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#6B7280',
    status VARCHAR(20) DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 修改 tickets 表：alert_source_id 改为可为空，添加 ticket_type_id
ALTER TABLE tickets
    MODIFY alert_source_id BIGINT UNSIGNED NULL,
    ADD COLUMN ticket_type_id BIGINT UNSIGNED NULL AFTER source_type;

-- 添加外键
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_ticket_type
    FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id);

-- 插入默认类型
INSERT INTO ticket_types (code, name, description, color) VALUES
    ('default', '默认', '未分类工单', '#6B7280'),
    ('network_fault', '网络故障', '网络设备或链路故障', '#EF4444'),
    ('server_alert', '服务器告警', '服务器性能或硬件告警', '#F59E0B');
