UPDATE data_source
SET port = '3306'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'MYSQL';

UPDATE data_source
SET port = '4000'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'TIDB';

UPDATE data_source
SET port = '2883'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'OCEANBASE';

UPDATE data_source
SET port = '9000'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'CLICKHOUSE';

UPDATE data_source
SET port = '5432'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'POSTGRES';

UPDATE data_source
SET port = '6379'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'REDIS';

UPDATE data_source
SET port = '5236'
FROM instance
WHERE data_source.instance_id = instance.id AND data_source.port = '' AND instance.engine = 'DM';