-- Rollback: Remove hierarchical categories support
-- This reverses migration 002_hierarchical_categories.sql

BEGIN;

-- Step 1: Drop the helper view
DROP VIEW IF EXISTS page_category_full;

-- Step 2: Drop the circular reference trigger and function
DROP TRIGGER IF EXISTS trg_check_category_circular ON categories;
DROP FUNCTION IF EXISTS check_category_circular_reference();

-- Step 3: Drop the slug format constraint
ALTER TABLE categories DROP CONSTRAINT IF EXISTS chk_slug_format;

-- Step 4: Drop indexes
DROP INDEX IF EXISTS idx_categories_path;
DROP INDEX IF EXISTS idx_categories_parent;

-- Step 5: Drop the path column
ALTER TABLE categories DROP COLUMN IF EXISTS path;

-- Step 6: Drop the parent_id column
ALTER TABLE categories DROP COLUMN IF EXISTS parent_id;

-- Note: We intentionally leave the ltree extension enabled as it's harmless
-- and may be used by other features. To remove it, run:
-- DROP EXTENSION IF EXISTS ltree;

COMMIT;
