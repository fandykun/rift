-- Migration: create_demo_projects | Created: 2026-06-23T09:05:00Z
CREATE TABLE demo_projects (
  id BIGSERIAL PRIMARY KEY,
  customer_id BIGINT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE demo_projects
  ADD CONSTRAINT demo_projects_customer_id_fk
  FOREIGN KEY (customer_id) REFERENCES demo_customers(id) NOT VALID;

CREATE INDEX demo_projects_customer_id_idx ON demo_projects (customer_id);

INSERT INTO demo_projects (customer_id, slug, name, status)
SELECT id, 'atlas', 'Atlas Billing Rewrite', 'active'
FROM demo_customers WHERE email = 'ada@example.com';

INSERT INTO demo_projects (customer_id, slug, name, status)
SELECT id, 'apollo', 'Apollo Data Warehouse', 'active'
FROM demo_customers WHERE email = 'grace@example.com';

INSERT INTO demo_projects (customer_id, slug, name, status)
SELECT id, 'kernel-lab', 'Kernel Lab Sandbox', 'paused'
FROM demo_customers WHERE email = 'linus@example.com';
