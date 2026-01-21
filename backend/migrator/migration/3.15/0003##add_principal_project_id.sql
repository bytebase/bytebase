-- Add project column to principal table for service accounts and workload identities.
-- This column references the project that owns the service account or workload identity.
-- For END_USER and SYSTEM_BOT types, this column should be NULL.
-- For SERVICE_ACCOUNT and WORKLOAD_IDENTITY types, this column indicates the owning project.
-- NULL project for SERVICE_ACCOUNT/WORKLOAD_IDENTITY means it's a workspace-level principal.

ALTER TABLE principal ADD COLUMN project text REFERENCES project(resource_id);

-- Create index for efficient lookups by project
CREATE INDEX idx_principal_project ON principal(project) WHERE project IS NOT NULL;

-- Add constraint to ensure END_USER and SYSTEM_BOT cannot have project
ALTER TABLE principal ADD CONSTRAINT principal_project_type_check
    CHECK (
        (type IN ('END_USER', 'SYSTEM_BOT') AND project IS NULL) OR
        (type IN ('SERVICE_ACCOUNT', 'WORKLOAD_IDENTITY'))
    );
