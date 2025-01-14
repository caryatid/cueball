CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE stage AS ENUM (
	'ENQUEUE',
	'INFLIGHT',
	'FAIL',
	'DONE'
);

-- append only table
CREATE TABLE execution_log (
	id UUID NOT NULL DEFAULT gen_random_uuid(), 
	stage stage,
	time TIMESTAMP DEFAULT now(),
	until TIMESTAMP DEFAULT NULL,
	worker TEXT,
	data JSONB
);

CREATE VIEW execution_state AS (
	SELECT id, stage, time, worker, data, until
	FROM (SELECT id, stage, time, worker, data, until, row_number()
	      OVER (PARTITION BY id ORDER BY time DESC) AS rn
	FROM execution_log) AS t
	WHERE rn = 1
);

