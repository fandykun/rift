-- Migration: create_demo_customers | Created: 2026-06-23T09:00:00Z
CREATE TABLE demo_customers (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  plan TEXT NOT NULL DEFAULT 'free',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX demo_customers_plan_idx ON demo_customers (plan);

INSERT INTO demo_customers (email, name, plan) VALUES
  ('ada@example.com', 'Ada Lovelace', 'team'),
  ('grace@example.com', 'Grace Hopper', 'enterprise'),
  ('linus@example.com', 'Linus Torvalds', 'free');
