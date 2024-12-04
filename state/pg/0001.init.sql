CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE stage AS ENUM (
	'RUNNING',
	'RETRY',
	'NEXT',
	'DONE'
);

-- append only table
CREATE TABLE execution_log (
	id UUID NOT NULL DEFAULT gen_random_uuid(), 
	stage stage,
	time TIMESTAMP DEFAULT current_timestamp(),
	worker TEXT,
	data JSONB
);

CREATE VIEW execution_state (
	SELECT id, stage, time, worker
	FROM (SELECT, id, stage, time, worker, row_number()
	      OVER (PARTITION BY id ORDER BY time DESC) AS rn
	FROM execution_log) AS t
	WHERE rn = 1
)

