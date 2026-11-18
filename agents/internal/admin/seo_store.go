package admin

import "context"

func (s *Store) migrateSEO(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS seo_briefs (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind TEXT NOT NULL DEFAULT 'page',
  url TEXT NOT NULL DEFAULT '',
  keyword TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  meta TEXT NOT NULL DEFAULT '',
  h1 TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  flags JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL DEFAULT 'proposed',
  rationale TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS seo_briefs_project_idx ON seo_briefs (project_id, created_at DESC);
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS pr_url TEXT NOT NULL DEFAULT '';
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS pack_token TEXT NOT NULL DEFAULT '';
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS inventory JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE seo_briefs ADD COLUMN IF NOT EXISTS plan_items JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE UNIQUE INDEX IF NOT EXISTS seo_briefs_pack_token_idx ON seo_briefs (pack_token) WHERE pack_token <> '';
CREATE TABLE IF NOT EXISTS seo_ga4 (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  property_id TEXT NOT NULL DEFAULT '',
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS seo_gsc (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  site_url TEXT NOT NULL DEFAULT '',
  snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}
