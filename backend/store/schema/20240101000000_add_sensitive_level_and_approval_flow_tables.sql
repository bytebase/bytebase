-- Copyright 2024 Bytebase Inc.
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Add sensitive levels table
CREATE TABLE IF NOT EXISTS sensitive_levels (
    id VARCHAR(255) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    level INT NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(255) NOT NULL,
    instance_id VARCHAR(255) NOT NULL,
    field_rules TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add approval flows table
CREATE TABLE IF NOT EXISTS approval_flows (
    id VARCHAR(255) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    sensitivity_level INT NOT NULL,
    steps TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add approval requests table
CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(255) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    issue_id VARCHAR(255) NOT NULL,
    sensitivity_level INT NOT NULL,
    approval_flow_id VARCHAR(255) NOT NULL,
    status INT NOT NULL,
    submitter VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_sensitive_levels_instance_id ON sensitive_levels(instance_id);
CREATE INDEX IF NOT EXISTS idx_sensitive_levels_schema_name ON sensitive_levels(schema_name);
CREATE INDEX IF NOT EXISTS idx_sensitive_levels_table_name ON sensitive_levels(table_name);
CREATE INDEX IF NOT EXISTS idx_sensitive_levels_level ON sensitive_levels(level);

CREATE INDEX IF NOT EXISTS idx_approval_flows_sensitivity_level ON approval_flows(sensitivity_level);

CREATE INDEX IF NOT EXISTS idx_approval_requests_issue_id ON approval_requests(issue_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_sensitivity_level ON approval_requests(sensitivity_level);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_approval_requests_submitter ON approval_requests(submitter);

-- Add foreign key constraints
ALTER TABLE approval_requests ADD CONSTRAINT fk_approval_requests_approval_flow_id FOREIGN KEY (approval_flow_id) REFERENCES approval_flows(id);