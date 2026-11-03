CREATE TABLE extraction_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    parser text NOT NULL,
    parser_version text NOT NULL,
    config jsonb NOT NULL DEFAULT '{}',
    identity_sha256 text,
    input_sha256 text,
    title text NOT NULL DEFAULT '',
    text text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(document_id, identity_sha256),
    UNIQUE(document_id, id)
);
CREATE INDEX extraction_runs_document_idx ON extraction_runs(document_id,created_at);
ALTER TABLE documents ADD COLUMN active_extraction_id uuid;
ALTER TABLE passages ADD COLUMN extraction_run_id uuid;
ALTER TABLE passages ADD COLUMN start_offset integer;
ALTER TABLE passages ADD COLUMN end_offset integer;

-- Preserve IDs and citations; provenance absent from old releases stays unknown.
INSERT INTO extraction_runs(document_id,parser,parser_version,config,input_sha256,title,text)
SELECT d.id,'legacy-unknown','unknown','{"legacy":true}',d.sha256,d.title,d.text
FROM documents d WHERE d.text<>'' OR EXISTS(SELECT 1 FROM passages p WHERE p.document_id=d.id);
UPDATE documents d SET active_extraction_id=e.id FROM extraction_runs e WHERE e.document_id=d.id;
UPDATE passages p SET extraction_run_id=d.active_extraction_id FROM documents d WHERE d.id=p.document_id;
ALTER TABLE passages ALTER COLUMN extraction_run_id SET NOT NULL;
ALTER TABLE passages ADD CONSTRAINT passages_extraction_fk
    FOREIGN KEY(document_id,extraction_run_id) REFERENCES extraction_runs(document_id,id) ON DELETE CASCADE;
ALTER TABLE documents ADD CONSTRAINT documents_active_extraction_fk
    FOREIGN KEY(id,active_extraction_id) REFERENCES extraction_runs(document_id,id) DEFERRABLE INITIALLY DEFERRED;
CREATE UNIQUE INDEX passages_extraction_ordinal_idx ON passages(extraction_run_id,ordinal);
ALTER TABLE passages ADD CONSTRAINT passages_offsets_check
    CHECK ((start_offset IS NULL AND end_offset IS NULL) OR
        (start_offset IS NOT NULL AND end_offset IS NOT NULL AND start_offset>=0 AND end_offset>=start_offset));
