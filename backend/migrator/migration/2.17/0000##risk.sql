UPDATE setting
SET value = REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(value, 'source == 6', 'source == \"DATA_EXPORT\"'), 'source == 5', 'source == \"REQUEST_EXPORT\"'), 'source == 4', 'source == \"REQUEST_QUERY\"'), 'source == 3', 'source == \"CREATE_DATABASE\"'), 'source == 2', 'source == \"DML\"'), 'source == 1', 'source == \"DDL\"')
WHERE name = 'bb.workspace.approval';