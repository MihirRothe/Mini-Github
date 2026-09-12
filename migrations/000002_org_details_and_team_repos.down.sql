-- ForgeHub Phase 3 Down Migration

DROP TABLE IF EXISTS team_repositories;
ALTER TABLE organizations DROP COLUMN IF EXISTS website;
ALTER TABLE organizations DROP COLUMN IF EXISTS location;
