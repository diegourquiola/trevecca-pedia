-- Rollback: feat/metadata → master
-- Removes metadata columns from revisions table and restores original foreign key constraints
-- WARNING: This will permanently delete the slug, name, archive_date, and deleted_at columns from revisions

BEGIN;

-- Step 1: Drop the new columns from revisions
ALTER TABLE revisions DROP COLUMN IF EXISTS slug;
ALTER TABLE revisions DROP COLUMN IF EXISTS name;
ALTER TABLE revisions DROP COLUMN IF EXISTS archive_date;
ALTER TABLE revisions DROP COLUMN IF EXISTS deleted_at;

-- Step 2: Restore original foreign key constraints (without CASCADE)

-- revisions.page_id - remove CASCADE
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'revisions' 
        AND tc.constraint_name LIKE '%page_id%'
        AND rc.delete_rule = 'CASCADE'
    ) THEN
        ALTER TABLE revisions DROP CONSTRAINT revisions_page_id_fkey;
        ALTER TABLE revisions ADD CONSTRAINT revisions_page_id_fkey 
            FOREIGN KEY (page_id) REFERENCES pages(uuid);
    END IF;
END $$;

-- snapshots.page - remove CASCADE
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'snapshots' 
        AND tc.constraint_name LIKE '%page%'
        AND rc.delete_rule = 'CASCADE'
    ) THEN
        ALTER TABLE snapshots DROP CONSTRAINT snapshots_page_fkey;
        ALTER TABLE snapshots ADD CONSTRAINT snapshots_page_fkey 
            FOREIGN KEY (page) REFERENCES pages(uuid);
    END IF;
END $$;

-- snapshots.revision - remove CASCADE
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'snapshots' 
        AND tc.constraint_name LIKE '%revision%'
        AND rc.delete_rule = 'CASCADE'
    ) THEN
        ALTER TABLE snapshots DROP CONSTRAINT snapshots_revision_fkey;
        ALTER TABLE snapshots ADD CONSTRAINT snapshots_revision_fkey 
            FOREIGN KEY (revision) REFERENCES revisions(uuid);
    END IF;
END $$;

-- page_categories.page_id - remove CASCADE
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'page_categories' 
        AND tc.constraint_name LIKE '%page_id%'
        AND rc.delete_rule = 'CASCADE'
    ) THEN
        ALTER TABLE page_categories DROP CONSTRAINT page_categories_page_id_fkey;
        ALTER TABLE page_categories ADD CONSTRAINT page_categories_page_id_fkey 
            FOREIGN KEY (page_id) REFERENCES pages(uuid);
    END IF;
END $$;

-- page_categories.category - remove CASCADE
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'page_categories' 
        AND tc.constraint_name LIKE '%category%'
        AND rc.delete_rule = 'CASCADE'
    ) THEN
        ALTER TABLE page_categories DROP CONSTRAINT page_categories_category_fkey;
        ALTER TABLE page_categories ADD CONSTRAINT page_categories_category_fkey 
            FOREIGN KEY (category) REFERENCES categories(id);
    END IF;
END $$;

-- Step 3: Restore pages.last_revision_id to original behavior (no SET NULL)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints tc
        JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE tc.table_name = 'pages' 
        AND tc.constraint_name = 'fk_pages_last_revision'
        AND rc.delete_rule = 'SET NULL'
    ) THEN
        ALTER TABLE pages DROP CONSTRAINT fk_pages_last_revision;
        ALTER TABLE pages ADD CONSTRAINT fk_pages_last_revision 
            FOREIGN KEY (last_revision_id) REFERENCES revisions(uuid);
    END IF;
END $$;

COMMIT;
