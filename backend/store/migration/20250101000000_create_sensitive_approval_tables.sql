-- Sensitive Levels table
CREATE TABLE IF NOT EXISTS sensitive_levels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    severity TEXT NOT NULL,
    description TEXT,
    color TEXT,
    field_match_rules TEXT,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL,
    update_time TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Approval Flows table
CREATE TABLE IF NOT EXISTS approval_flows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    sensitive_severity TEXT NOT NULL,
    steps TEXT,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL,
    update_time TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Approval Requests table
CREATE TABLE IF NOT EXISTS approval_requests (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    requester TEXT NOT NULL,
    status TEXT NOT NULL,
    sensitive_severity TEXT,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL,
    update_time TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Approval Request Details table
CREATE TABLE IF NOT EXISTS approval_request_details (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    approval_request_name TEXT NOT NULL,
    sensitive_levels TEXT,
    database_instance TEXT NOT NULL,
    database TEXT NOT NULL,
    table_name TEXT NOT NULL,
    field_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    sql_statement TEXT NOT NULL,
    FOREIGN KEY (approval_request_name) REFERENCES approval_requests(name)
);

-- Approval Logs table
CREATE TABLE IF NOT EXISTS approval_logs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    approval_request_name TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    approver TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT,
    comment TEXT,
    create_time TIMESTAMP WITH TIME ZONE NOT NULL,
    FOREIGN KEY (approval_request_name) REFERENCES approval_requests(name)
);

-- Sensitive Data Mapping table
CREATE TABLE IF NOT EXISTS sensitive_data_mapping (
    id TEXT PRIMARY KEY,
    sensitive_level_id TEXT NOT NULL,
    database_instance TEXT NOT NULL,
    database TEXT NOT NULL,
    table_name TEXT NOT NULL,
    field_name TEXT NOT NULL,
    field_type TEXT,
    mapping_type TEXT NOT NULL,
    FOREIGN KEY (sensitive_level_id) REFERENCES sensitive_levels(id),
    UNIQUE (database_instance, database, table_name, field_name)
);

-- Default data for sensitive levels
INSERT INTO sensitive_levels (id, name, display_name, severity, description, color, field_match_rules, create_time, update_time)
VALUES 
('high', 'sensitive-levels/high', 'High Sensitivity', 'SEVERITY_HIGH', 'High sensitivity data that requires strict protection', '#FF0000', '[]', NOW(), NOW()),
('medium', 'sensitive-levels/medium', 'Medium Sensitivity', 'SEVERITY_MEDIUM', 'Medium sensitivity data', '#FFA500', '[]', NOW(), NOW()),
('low', 'sensitive-levels/low', 'Low Sensitivity', 'SEVERITY_LOW', 'Low sensitivity data', '#00FF00', '[]', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Default data for approval flows
INSERT INTO approval_flows (id, name, display_name, description, sensitive_severity, steps, create_time, update_time)
VALUES 
('high-flow', 'approval-flows/high-flow', 'High Sensitive Approval Flow', 'Requires 2 levels of approval: Department Head -> DBA', 'SEVERITY_HIGH', '[{"display_name":"Department Head Review","role":"DEPARTMENT_HEAD","approver_type":"ROLE"},{"display_name":"DBA Review","role":"DBA","approver_type":"ROLE"}]', NOW(), NOW()),
('medium-flow', 'approval-flows/medium-flow', 'Medium Sensitive Approval Flow', 'Requires 1 level of approval: DBA', 'SEVERITY_MEDIUM', '[{"display_name":"DBA Review","role":"DBA","approver_type":"ROLE"}]', NOW(), NOW()),
('low-flow', 'approval-flows/low-flow', 'Low Sensitive Approval Flow', 'No approval required', 'SEVERITY_LOW', '[]', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
