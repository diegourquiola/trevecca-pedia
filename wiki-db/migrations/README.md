# Database Migration Guide: master → feat/metadata

## Overview
This migration adds metadata columns to the `revisions` table and updates foreign key constraints to use `ON DELETE CASCADE`.

## Prerequisites
- Access to the Fly.io database
- The database has existing pages with single (creation) revisions

## Migration Steps

### Option 1: Using fly pg (Recommended for Fly.io Postgres)

1. Connect to your database:
```bash
fly pg connect --app <your-postgres-app-name>
```

2. Execute the migration:
```sql
\i wiki-db/migrations/001_master_to_metadata.sql
```

Or paste the SQL content directly.

### Option 2: Using psql with connection string

```bash
# Export your DATABASE_URL
export DATABASE_URL="postgres://user:pass@host:port/dbname"

# Run the migration
psql $DATABASE_URL -f wiki-db/migrations/001_master_to_metadata.sql
```

### Option 3: Using fly ssh (for apps with database access)

1. SSH into the wiki app:
```bash
fly ssh console --app <your-wiki-app-name>
```

2. Run psql with the migration (if psql is available in the container):
```bash
psql $DATABASE_URL -f /path/to/001_master_to_metadata.sql
```

## Verification

After migration, verify the changes:

```sql
-- Check new columns exist
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'revisions';

-- Expected columns: uuid, page_id, date_time, author, slug, name, archive_date, deleted_at

-- Check foreign key constraints have CASCADE
SELECT tc.constraint_name, rc.delete_rule
FROM information_schema.table_constraints tc
JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
WHERE tc.table_name IN ('revisions', 'snapshots', 'page_categories')
AND tc.constraint_type = 'FOREIGN KEY';

-- Expected: all delete_rule should be 'CASCADE' (except pages.last_revision_id which is 'SET NULL')

-- Verify revisions have populated slug and name
SELECT COUNT(*) as total_revisions,
       COUNT(slug) as revisions_with_slug,
       COUNT(name) as revisions_with_name
FROM revisions;
```

## Rollback

If you need to rollback, run:

```bash
psql $DATABASE_URL -f wiki-db/migrations/rollback_001_metadata_to_master.sql
```

⚠️ **Warning**: Rollback will permanently delete the `slug`, `name`, `archive_date`, and `deleted_at` columns from the revisions table. Any data in these columns will be lost.

## Post-Migration

After successful migration:
1. Deploy the new application code (feat/metadata branch)
2. The new columns will store historical metadata for each revision
3. The `slug` and `name` in revisions track the page title/slug at the time of each revision
4. `archive_date` and `deleted_at` support future soft-delete and archival features

## Troubleshooting

### Error: "column does not exist" during population
If the population step fails because some revisions don't have corresponding pages, check:
```sql
SELECT r.uuid, r.page_id 
FROM revisions r 
LEFT JOIN pages p ON r.page_id = p.uuid 
WHERE p.uuid IS NULL;
```

### Error: "violates not-null constraint"
If you get NOT NULL constraint errors after adding columns but before populating, the migration script order is wrong. The script should populate BEFORE making columns NOT NULL.

## Files

- `001_master_to_metadata.sql` - Migration script (idempotent, safe to run multiple times)
- `rollback_001_metadata_to_master.sql` - Rollback script (destructive - removes columns)
