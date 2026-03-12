-- Migration: Add hierarchical categories support
-- Adds ltree extension, parent_id, path columns, circular reference prevention
-- This migration is idempotent and safe to run multiple times

BEGIN;

-- Step 1: Enable ltree extension (safe to run multiple times)
CREATE EXTENSION IF NOT EXISTS ltree;

-- Step 2: Add parent_id column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'categories' AND column_name = 'parent_id') THEN
        ALTER TABLE categories ADD COLUMN parent_id INTEGER REFERENCES categories(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Step 3: Add path column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'categories' AND column_name = 'path') THEN
        ALTER TABLE categories ADD COLUMN path LTREE NOT NULL DEFAULT 'root';
    END IF;
END $$;

-- Step 4: Add indexes (idempotent - safe to recreate)
DROP INDEX IF EXISTS idx_categories_path;
CREATE INDEX idx_categories_path ON categories USING GIST(path);

DROP INDEX IF EXISTS idx_categories_parent;
CREATE INDEX idx_categories_parent ON categories(parent_id);

-- Step 5: Update existing categories with paths (only if they have null/empty paths or no path yet)
-- This sets paths for existing root categories (those without parent_id)
UPDATE categories 
SET path = ('root.' || slug)::ltree
WHERE parent_id IS NULL 
AND (path IS NULL OR path = 'root');

-- Step 6: Add slug format constraint (check if exists first)
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.constraint_column_usage 
        WHERE table_name = 'categories' AND constraint_name = 'chk_slug_format'
    ) THEN
        ALTER TABLE categories 
        ADD CONSTRAINT chk_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$');
    END IF;
END $$;

-- Step 7: Create/update circular reference prevention trigger function
CREATE OR REPLACE FUNCTION check_category_circular_reference()
RETURNS TRIGGER AS $$
BEGIN
    -- Prevent a category from being its own parent
    IF NEW.parent_id = NEW.id THEN
        RAISE EXCEPTION 'Category cannot be its own parent';
    END IF;
    
    -- Prevent circular references via path check
    IF NEW.parent_id IS NOT NULL THEN
        IF EXISTS (
            SELECT 1 FROM categories 
            WHERE id = NEW.parent_id 
            AND NEW.path @> path
        ) THEN
            RAISE EXCEPTION 'Circular reference detected: parent is already a descendant';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 8: Create trigger (drop if exists first)
DROP TRIGGER IF EXISTS trg_check_category_circular ON categories;
CREATE TRIGGER trg_check_category_circular
BEFORE INSERT OR UPDATE ON categories
FOR EACH ROW EXECUTE FUNCTION check_category_circular_reference();

-- Step 9: Create helper view for page categories with full paths
DROP VIEW IF EXISTS page_category_full;
CREATE VIEW page_category_full AS
SELECT 
    pc.page_id,
    c.id AS category_id,
    c.slug AS category_slug,
    c.name AS category_name,
    c.path,
    c.parent_id,
    CASE 
        WHEN c.parent_id IS NULL THEN c.slug
        ELSE (SELECT slug FROM categories WHERE id = c.parent_id) || '/' || c.slug
    END AS full_slug
FROM page_categories pc
JOIN categories c ON pc.category = c.id;

COMMIT;
