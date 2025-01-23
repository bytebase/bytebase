UPDATE data_source dst
SET host = src.host, port = src.port
FROM data_source src
WHERE dst.database_id = src.database_id
AND dst.type = 'RO'
AND src.type = 'ADMIN';
