# Database Migration Guide: Add Moderator Role

## Overview
This migration adds the 'moderator' role to the roles table to enable future moderation features.

## Migration Steps

### Option 1: Using fly pg (Recommended for Fly.io Postgres)

1. Connect to your database:
```bash
fly pg connect --app <your-postgres-app-name>
```

2. Execute the migration:
```sql
\i auth-db/migrations/001_add_moderator_role.sql
```

Or paste the SQL content directly.

### Option 2: Using psql with connection string

```bash
# Export your DATABASE_URL
export DATABASE_URL="postgres://user:pass@host:port/dbname"

# Run the migration
psql $DATABASE_URL -f auth-db/migrations/001_add_moderator_role.sql
```

### Option 3: Using fly ssh (for apps with database access)

1. SSH into the auth app:
```bash
fly ssh console --app <your-auth-app-name>
```

2. Run psql with the migration (if psql is available in the container):
```bash
psql $DATABASE_URL -f /path/to/001_add_moderator_role.sql
```

## Verification

After migration, verify the moderator role exists:

```sql
SELECT * FROM roles WHERE name = 'moderator';
```

Expected output:
```
 id |   name
----+-----------
  4 | moderator
```

## Rollback

To remove the moderator role (not recommended if any users have been assigned this role):

```sql
DELETE FROM roles WHERE name = 'moderator';
```

⚠️ **Warning**: Only rollback if no users have been assigned the moderator role. Check first:
```sql
SELECT COUNT(*) as moderator_count 
FROM user_roles ur 
JOIN roles r ON ur.role_id = r.id 
WHERE r.name = 'moderator';
```

## Post-Migration

After successful migration:
1. The 'moderator' role is available for assignment to users
2. New users continue to be assigned 'contributor' role by default
3. Future features can check for the 'moderator' role in JWT tokens

## Files

- `001_add_moderator_role.sql` - Migration script (idempotent, safe to run multiple times)
