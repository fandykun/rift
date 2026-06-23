-- Migration: add_demo_deployment_audit | Created: 2026-06-23T09:10:00Z
CREATE TABLE demo_deployment_audit (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL,
  migration_version TEXT NOT NULL,
  deployed_by TEXT NOT NULL,
  deployed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  result TEXT NOT NULL DEFAULT 'success'
);

ALTER TABLE demo_deployment_audit
  ADD CONSTRAINT demo_deployment_audit_project_id_fk
  FOREIGN KEY (project_id) REFERENCES demo_projects(id) NOT VALID;

CREATE INDEX demo_deployment_audit_project_id_idx ON demo_deployment_audit (project_id);

INSERT INTO demo_deployment_audit (project_id, migration_version, deployed_by, result)
SELECT id, '20260623_090000', 'demo@rift.dev', 'success'
FROM demo_projects WHERE slug = 'atlas';

INSERT INTO demo_deployment_audit (project_id, migration_version, deployed_by, result)
SELECT id, '20260623_090500', 'demo@rift.dev', 'success'
FROM demo_projects WHERE slug = 'apollo';
