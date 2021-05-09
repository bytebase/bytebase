-- Insert a second workspace for testing purpose to make sure we don't
-- advertently return data from a different workspace.
INSERT INTO
    workspace (
        creator_id,
        updater_id,
        row_status,
        slug,
        name
    )
VALUES
    (1, 1, 'NORMAL', 'ws2', 'Workspace2');