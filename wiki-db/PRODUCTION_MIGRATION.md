# Production Database Migration: Hierarchical Categories

## Overview

This document describes how to apply migration 002_hierarchical_categories.sql to the production fly.io database.

**Status**: The production database currently has no existing categories, making this migration straightforward.

## Prerequisites

- fly CLI installed and authenticated
- Database app: `trevecca-pedia-db`
- Database name: `trevecca_pedia_wiki`
- Access to the `002_hierarchical_categories.sql` migration file

## Migration Steps

### Step 1: Verify Database Connectivity

```bash
# Check fly authentication
fly auth whoami

# Verify database app is running
fly status --app trevecca-pedia-db
```

### Step 2: Apply the Migration

**IMPORTANT**: This migration adds the ltree extension and modifies the categories table. Since there are no existing categories, this is a non-destructive operation.

```bash
# Navigate to wiki-db directory
cd wiki-db

# Apply the migration using fly postgres connect
printf '%s\n\\q\n' "$(cat migrations/002_hierarchical_categories.sql)" | fly postgres connect --app trevecca-pedia-db --database trevecca_pedia_wiki
```

### Step 3: Verify Migration Success

Connect to the database and verify the migration was applied:

```bash
# Connect to database
fly postgres connect --app trevecca-pedia-db --database trevecca_pedia_wiki

# Run verification queries:

-- Check ltree extension is enabled
SELECT * FROM pg_extension WHERE extname = 'ltree';

-- Check new columns exist
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'categories' 
AND column_name IN ('parent_id', 'path');

-- Check indexes were created
SELECT indexname 
FROM pg_indexes 
WHERE tablename = 'categories' 
AND indexname IN ('idx_categories_path', 'idx_categories_parent');

-- Check trigger exists
SELECT tgname 
FROM pg_trigger 
WHERE tgname = 'trg_check_category_circular';

-- Check view exists
SELECT * FROM pg_views WHERE viewname = 'page_category_full';

-- Exit psql
\q
```

### Step 4: Rollback (if needed)

If you need to rollback the migration:

```bash
printf '%s\n\\q\n' "$(cat migrations/rollback_002_hierarchical_categories.sql)" | fly postgres connect --app trevecca-pedia-db --database trevecca_pedia_wiki
```

## Migration Details

### What This Migration Does

1. **Enables ltree extension** - PostgreSQL extension for hierarchical data
2. **Adds parent_id column** - INTEGER with foreign key to categories(id) ON DELETE CASCADE
3. **Adds path column** - LTREE type with GIST index for efficient tree queries
4. **Creates indexes** - GIST index on path, B-tree index on parent_id
5. **Adds slug format constraint** - Enforces lowercase letters, numbers, and hyphens only
6. **Creates circular reference trigger** - Prevents a category from being its own parent or creating circular references
7. **Creates page_category_full view** - Helper view for joining page_categories with full category paths

### Error Handling

The migration includes these checks:
- **Category cannot be its own parent** - Error code P0001 with message 'Category cannot be its own parent'
- **Circular reference detected** - Error code P0001 with message 'Circular reference detected: parent is already a descendant'
- **Invalid slug format** - Constraint violation for invalid characters in slug

These errors should be handled in the Go application layer when creating or updating categories.

## Post-Migration

After migration is complete, you can:

1. Create hierarchical categories using the new schema
2. Assign pages to subcategories
3. Query pages by parent category (includes descendants) or exact match only

### Example Usage

```sql
-- Create root category
INSERT INTO categories (slug, name, path) 
VALUES ('people', 'People', 'root.people');

-- Create subcategory
INSERT INTO categories (slug, name, parent_id, path) 
SELECT 'faculty', 'Faculty', id, 'root.people.faculty'
FROM categories WHERE slug = 'people';

-- Query all categories in People tree (includes faculty, staff, etc.)
SELECT * FROM categories WHERE path <@ 'root.people';
```

## Troubleshooting

### Issue: "extension ltree is not available"

The ltree extension is included in standard PostgreSQL installations. If not available:

```bash
# For fly.io, this is typically already available
# If not, you may need to provision a new database with extension support
```

### Issue: Migration fails midway

The migration is wrapped in a transaction. If it fails, all changes will be rolled back. You can safely re-run it.

### Issue: View or trigger already exists

The migration uses `DROP IF EXISTS` for views and triggers, so it's safe to re-run if some components were partially applied.

## Contact

If issues arise during migration, check the following:
1. Verify fly CLI authentication: `fly auth whoami`
2. Check database status: `fly status --app trevecca-pedia-db`
3. Review migration logs for specific error messages
