package index

const schemaVersion = 1

const schemaSQL = `
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS index_meta (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS artifacts (
	id TEXT PRIMARY KEY,
	session_id TEXT,
	task_id TEXT,
	run_id TEXT,
	type TEXT,
	created_at INTEGER,
	text TEXT,
	provenance_json TEXT,
	artifact_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_artifacts_session ON artifacts(session_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_task ON artifacts(task_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts(run_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type);
CREATE INDEX IF NOT EXISTS idx_artifacts_created ON artifacts(created_at);
CREATE TABLE IF NOT EXISTS artifact_links (
	artifact_id TEXT NOT NULL,
	link_type TEXT NOT NULL,
	link_value TEXT NOT NULL,
	FOREIGN KEY(artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_links_value ON artifact_links(link_type, link_value);
CREATE INDEX IF NOT EXISTS idx_links_artifact ON artifact_links(artifact_id);
CREATE VIRTUAL TABLE IF NOT EXISTS artifacts_fts USING fts5(
	text,
	content='artifacts',
	content_rowid='rowid',
	tokenize='porter'
);
`

const metaKeySchemaVersion = "schema_version"
