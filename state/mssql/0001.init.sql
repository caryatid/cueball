CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

/*
CREATE TYPE stage AS ENUM (
	'RUNNING',
	'RETRY',
	'NEXT',
	'DONE'
);
*/ TODO NO ENUMS in MSSQL - use join table

-- append only table
CREATE TABLE execution_log (
	id uniqueidentifier NOT NULL DEFAULT NEWID(), 
	stage string, -- TODO make enum
	time DATETIME2 DEFAULT SYSUTCDATETIME(),
	worker JSON
);

CREATE VIEW execution_state (
	SELECT id, stage, time, worker
	FROM (SELECT, id, stage, time, worker, row_number()
	      OVER (PARTITION BY id ORDER BY time DESC) AS rn
	FROM execution_log)
	WHERE rn = 1
)

