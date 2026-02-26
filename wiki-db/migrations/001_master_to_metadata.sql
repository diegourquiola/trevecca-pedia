-- Migration: master → feat/metadata
-- Adds metadata columns to revisions table and updates foreign key constraints
-- This migration is idempotent and safe to run multiple times

BEGIN;

-- Step 1: Add new columns to revisions table if they don't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'revisions' AND column_name = 'slug') THEN
        ALTER TABLE revisions ADD COLUMN slug TEXT;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'revisions' AND column_name = 'name') THEN
        ALTER TABLE revisions ADD COLUMN name TEXT;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'revisions' AND column_name = 'archive_date') THEN
        ALTER TABLE revisions ADD COLUMN archive_date DATE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'revisions' AND column_name = 'deleted_at') THEN
        ALTER TABLE revisions ADD COLUMN deleted_at TIMESTAMP;
    END IF;
END $$;

-- Step 2: Populate slug and name from corresponding pages (only for the initial/creation revisions)
-- Since all pages have only one revision (creation), we can safely copy from pages
UPDATE revisions r
SET 
    slug = p.slug,
    name = p.name
FROM pages p
WHERE r.page_id = p.uuid
  AND (r.slug IS NULL OR r.slug = '');

-- Step 3: Make columns NOT NULL after population
ALTER TABLE revisions ALTER COLUMN slug SET NOT NULL;
ALTER TABLE revisions ALTER COLUMN name SET NOT NULL;

-- Step 4: Update foreign key constraints to add ON DELETE CASCADE
-- First, drop and recreate revisions.page_id FK with CASCADE
DO $$ 
BEGIN
    -- Check if the constraint exists with different cascade behavior
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'revisions' 
        AND tc.constraint_name LIKE '%page_id%'
        AND rc.delete_rule != 'CASCADE'
    ) THEN
        -- Get the constraint name
        ALTER TABLE revisions DROP CONSTRAINT revisions_page_id_fkey;
        ALTER TABLE revisions ADD CONSTRAINT revisions_page_id_fkey 
            FOREIGN KEY (page_id) REFERENCES pages(uuid) ON DELETE CASCADE;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'revisions' AND constraint_name LIKE '%page_id%'
    ) THEN
        -- Constraint doesn't exist, add it
        ALTER TABLE revisions ADD CONSTRAINT revisions_page_id_fkey 
            FOREIGN KEY (page_id) REFERENCES pages(uuid) ON DELETE CASCADE;
    END IF;
END $$;

-- Step 5: Update snapshots foreign key constraints
DO $$ 
BEGIN
    -- snapshots.page FK
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'snapshots' 
        AND tc.constraint_name LIKE '%page%'
        AND rc.delete_rule != 'CASCADE'
    ) THEN
        ALTER TABLE snapshots DROP CONSTRAINT snapshots_page_fkey;
        ALTER TABLE snapshots ADD CONSTRAINT snapshots_page_fkey 
            FOREIGN KEY (page) REFERENCES pages(uuid) ON DELETE CASCADE;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'snapshots' AND constraint_name LIKE '%page%'
    ) THEN
        ALTER TABLE snapshots ADD CONSTRAINT snapshots_page_fkey 
            FOREIGN KEY (page) REFERENCES pages(uuid) ON DELETE CASCADE;
    END IF;

    -- snapshots.revision FK
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'snapshots' 
        AND tc.constraint_name LIKE '%revision%'
        AND rc.delete_rule != 'CASCADE'
    ) THEN
        ALTER TABLE snapshots DROP CONSTRAINT snapshots_revision_fkey;
        ALTER TABLE snapshots ADD CONSTRAINT snapshots_revision_fkey 
            FOREIGN KEY (revision) REFERENCES revisions(uuid) ON DELETE CASCADE;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'snapshots' AND constraint_name LIKE '%revision%'
    ) THEN
        ALTER TABLE snapshots ADD CONSTRAINT snapshots_revision_fkey 
            FOREIGN KEY (revision) REFERENCES revisions(uuid) ON DELETE CASCADE;
    END IF;
END $$;

-- Step 6: Update page_categories foreign key constraints
DO $$ 
BEGIN
    -- page_categories.page_id FK
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'page_categories' 
        AND tc.constraint_name LIKE '%page_id%'
        AND rc.delete_rule != 'CASCADE'
    ) THEN
        ALTER TABLE page_categories DROP CONSTRAINT page_categories_page_id_fkey;
        ALTER TABLE page_categories ADD CONSTRAINT page_categories_page_id_fkey 
            FOREIGN KEY (page_id) REFERENCES pages(uuid) ON DELETE CASCADE;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'page_categories' AND constraint_name LIKE '%page_id%'
    ) THEN
        ALTER TABLE page_categories ADD CONSTRAINT page_categories_page_id_fkey 
            FOREIGN KEY (page_id) REFERENCES pages(uuid) ON DELETE CASCADE;
    END IF;

    -- page_categories.category FK
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'page_categories' 
        AND tc.constraint_name LIKE '%category%'
        AND rc.delete_rule != 'CASCADE'
    ) THEN
        ALTER TABLE page_categories DROP CONSTRAINT page_categories_category_fkey;
        ALTER TABLE page_categories ADD CONSTRAINT page_categories_category_fkey 
            FOREIGN KEY (category) REFERENCES categories(id) ON DELETE CASCADE;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'page_categories' AND constraint_name LIKE '%category%'
    ) THEN
        ALTER TABLE page_categories ADD CONSTRAINT page_categories_category_fkey 
            FOREIGN KEY (category) REFERENCES categories(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Step 7: Update pages.last_revision_id FK to use ON DELETE SET NULL (feat/metadata behavior)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'pages' 
        AND tc.constraint_name = 'fk_pages_last_revision'
        AND rc.delete_rule != 'SET NULL'
    ) THEN
        ALTER TABLE pages DROP CONSTRAINT fk_pages_last_revision;
        ALTER TABLE pages ADD CONSTRAINT fk_pages_last_revision 
            FOREIGN KEY (last_revision_id) REFERENCES revisions(uuid) ON DELETE SET NULL;
    END IF;
END $$;

COMMIT;
