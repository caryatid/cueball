CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE stage AS ENUM (
	'ENQUEUE',
	'INFLIGHT',
	'DONE'
);

-- append only table
CREATE TABLE execution_log (
	id UUID NOT NULL DEFAULT gen_random_uuid(), 
	stage stage,
	time TIMESTAMP DEFAULT now(),
	worker TEXT,
	data JSONB
);

CREATE VIEW execution_state AS (
	SELECT id, stage, time, worker, data
	FROM (SELECT id, stage, time, worker, data, row_number()
	      OVER (PARTITION BY id ORDER BY time DESC) AS rn
	FROM execution_log) AS t
	WHERE rn = 1
);

