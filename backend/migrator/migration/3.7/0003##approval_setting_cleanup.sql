UPDATE setting SET value = replace(value::text, E'\\n', ' ') WHERE name = 'bb.workspace.approval';

UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 300 || ', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 200 || ', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 100 || ', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 0 || ', '') WHERE name = 'bb.workspace.approval';

UPDATE setting SET value = replace(value::text, ' || source == \"REQUEST_EXPORT\" && level == 300', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, ' || source == \"REQUEST_EXPORT\" && level == 200', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, ' || source == \"REQUEST_EXPORT\" && level == 100', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, ' || source == \"REQUEST_EXPORT\" && level == 0', '') WHERE name = 'bb.workspace.approval';

UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 300', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 200', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 100', '') WHERE name = 'bb.workspace.approval';
UPDATE setting SET value = replace(value::text, 'source == \"REQUEST_EXPORT\" && level == 0', '') WHERE name = 'bb.workspace.approval';

UPDATE setting SET value = replace(value::text, 'REQUEST_QUERY', 'REQUEST_ROLE') WHERE name = 'bb.workspace.approval';
