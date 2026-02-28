# Trevecca Pedia Deployment Guide

This guide covers deploying the Trevecca Pedia application to fly.io with persistent storage volumes.

## Architecture Overview

```
    External          ┌──────────────────┐
    Access            │   Web Frontend   │
    (Users)           │ (trevecca-pedia- │
         │            │      web)        │
         │            └────────┬─────────┘
         │                     │ HTTP
         │            ┌────────▼─────────┐
         └───────────▶│   API Layer      │
                      │ (trevecca-pedia- │
                      │      api)        │
                      └────────┬─────────┘
                               │
                 ┌─────────────┼──────────┐
                 │                        │ 
                 ▼                        ▼           
        ┌─────────────────┐      ┌─────────────────┐        ┌─────────────────┐
        │  Wiki Service   │      │  Search Service │        │  Search Index   │
        │ (trevecca-pedia-│      │ (trevecca-pedia-│──────▶ │    Volume       │
        │      wiki)      │      │     search)     │        │    (/index)     │
        └────────┬────────┘      └─────────────────┘        └─────────────────┘
                 │
       ┌─────────┴──────────┐
       ▼                    ▼
┌──────────────┐   ┌─────────────────┐
│  Postgres DB │   │  Wiki Files     │
│(trevecca-    │   │  Volume (/data) │
│  pedia-db)   │   └─────────────────┘
└──────────────┘
```

**Service Communication:**
- **Web Frontend** → API Layer (only)
- **API Layer** → Wiki Service, Search Service (routes/proxies requests)
- **Wiki Service** → Postgres DB, Wiki Files Volume (only service with DB access)
- **Search Service** → Search Index Volume (only service with index access)

## Prerequisites

- [flyctl CLI](https://fly.io/docs/hands-on/install-flyctl/) installed
- Logged into fly.io: `fly auth login`
- Go 1.25+ (for local testing)

## Deployment Order

Deploy in this order to avoid circular dependencies:

1. **Wiki Service** (wiki) - Deploy first; will receive database URL when attached
2. **Database** (wiki-db) - Create and attach to wiki service
3. **Auth Service** (auth) - Deploy first; will receive database URL when attached
4. **Auth Database** (auth-db) - Attach shared Postgres app and apply schema
5. **API Layer** (api-layer) - Deploy with wiki + auth URLs only initially
6. **Search Service** (search) - Depends on API layer for data (can now fetch)
7. **API Layer** (api-layer) - Redeploy with search URL added
8. **Web Frontend** (web) - Depends on fully configured API layer

**Why this order?** 
- The wiki service uses `DATABASE_URL` which is automatically set when the database is attached via `fly postgres attach`. This allows us to deploy the wiki service first without needing database credentials upfront.
- The auth service uses `DATABASE_URL` the same way. After attaching Postgres, apply the auth schema (tables + whitelist).
- The search service needs the API layer URL to fetch page data, but the API layer also needs the search service URL to proxy requests. By deploying the API layer twice (steps 5 and 7), we break this circular dependency. The web frontend is deployed last as it's stateless and only needs the public API layer URL.

## 1. Deploy Wiki Service

```bash
cd wiki

# Create the app
fly apps create trevecca-pedia-wiki

# Create volume for file storage (1GB)
fly volumes create wiki_data --region iad --size 1 --app trevecca-pedia-wiki

# Deploy
fly deploy
```

**Note:** The wiki service will start but won't be able to handle requests until the database is attached in step 2. This is expected behavior.

**Volume Details:**
- **Name:** `wiki_data`
- **Mount Point:** `/data`
- **Contents:** `pages/`, `revisions/`, `snapshots/`
- **Size:** 1GB
- **Region:** iad

## 2. Deploy Database

For production with automatic failover, create 2 machines (primary + replica):

```bash
cd wiki-db

# Create Postgres cluster with 2 machines for HA
fly postgres create \
  --name trevecca-pedia-db \
  --region iad \
  --vm-size shared-cpu-1x \
  --volume-size 1 \
  --initial-cluster-size 2

# Attach database to wiki service (sets DATABASE_URL automatically)
fly postgres attach trevecca-pedia-db --app trevecca-pedia-wiki

# Apply schema (without seed data)
./setup-db.sh trevecca-pedia-db trevecca-pedia-wiki

# Verify cluster health
fly status --app trevecca-pedia-db
```

**Note:** The 2-machine setup provides automatic failover. If the primary fails, the replica takes over. All connection strings use the internal hostname which automatically routes to the current primary. The `fly postgres attach` command automatically sets the `DATABASE_URL` secret on the wiki service.

## 3. Deploy Auth Service

Deploy the auth service first. Like the wiki service, it expects `DATABASE_URL` to be provided by Fly after Postgres is attached.

```bash
cd auth

# Create the app
fly apps create trevecca-pedia-auth

# Set required secrets
# IMPORTANT: This must match the JWT_SECRET used by api-layer
# Use this to get a string for the secret:
# openssl rand -base64 32
fly secrets set JWT_SECRET="<shared-jwt-secret>" --app trevecca-pedia-auth

# Deploy
fly deploy
```

**Note:** The auth service will start but won't be able to handle requests until the database is attached in step 4.

## 4. Configure Auth Database

The auth database schema lives in `auth-db/init/` and is applied to Fly Postgres (same pattern as `wiki-db/`).

This guide assumes the shared Postgres app is `trevecca-pedia-db` (created in step 2).

```bash
# Attach Postgres (creates the database + sets DATABASE_URL on auth app)
fly postgres attach trevecca-pedia-db --app trevecca-pedia-auth

# Apply auth schema (tables + whitelist)
cd ../auth-db
chmod +x setup-db.sh
./setup-db.sh trevecca-pedia-db trevecca_pedia_auth
```

## 5. Deploy API Layer (Initial)

**First deployment** - configure wiki + auth service URLs only. Search service URL will be added in step 7.

```bash
cd api-layer

# Create the app
fly apps create trevecca-pedia-api

# Set wiki service URL only (external fly.io HTTP address)
fly secrets set WIKI_SERVICE_URL="https://trevecca-pedia-wiki.fly.dev" --app trevecca-pedia-api

# Set auth service URL (external fly.io HTTP address)
fly secrets set AUTH_SERVICE_URL="https://trevecca-pedia-auth.fly.dev" --app trevecca-pedia-api

# Set JWT_SECRET so api-layer can validate tokens
# IMPORTANT: Must match auth service JWT_SECRET
fly secrets set JWT_SECRET="<shared-jwt-secret>" --app trevecca-pedia-api

# Deploy
fly deploy
```

**Note:** The API layer will start but search functionality won't work yet. We'll add the search service URL in step 7 after the search service is deployed.

## 6. Deploy Search Service

Now that the API layer is running, we can deploy the search service which needs the API layer URL to fetch page data.

```bash
cd search

# Create the app
fly apps create trevecca-pedia-search

# Create volume for search index (1GB)
fly volumes create search_index --region iad --size 1 --app trevecca-pedia-search

# Set API layer URL (now available from step 3)
fly secrets set API_LAYER_URL="https://trevecca-pedia-api.fly.dev/v1" --app trevecca-pedia-search

# Deploy
fly deploy
```

**Volume Details:**
- **Name:** `search_index`
- **Mount Point:** `/index`
- **Contents:** Bleve search index files
- **Size:** 1GB
- **Region:** iad

**Note:** On first startup, the search service will automatically fetch all pages from the API layer and build the search index. This may take a few minutes depending on the number of pages.

## 7. Deploy API Layer (Final - with Search URL)

**Second deployment** - add the search service URL so the API layer can proxy search requests.

```bash
cd api-layer

# Add search service URL
fly secrets set SEARCH_SERVICE_URL="https://trevecca-pedia-search.fly.dev" --app trevecca-pedia-api

# Redeploy to pick up the new configuration
fly deploy
```

**Done!** All services are now fully configured and communicating properly.

## 8. Deploy Web Frontend

The web frontend is a stateless Go application with Tailwind CSS that serves the user interface.

```bash
cd web

# Create the app
fly apps create trevecca-pedia-web

# Set API layer URL (public URL for the web frontend to use)
fly secrets set API_LAYER_URL="https://trevecca-pedia-api.fly.dev/v1" --app trevecca-pedia-web

# Deploy
fly deploy
```

**Notes:**
- The web service is **stateless** (no persistent volume needed)
- CSS is built automatically during the Docker build process
- Static files and templates are included in the container
- The web frontend is the public-facing entry point for users

## Volume Backup Strategy

**Important:** Fly.io volumes are persistent but not automatically backed up. Data loss can occur if the volume is deleted or corrupted.

### Automated Backups with GitHub Actions

A GitHub Actions workflow automatically creates daily snapshots at 2 AM UTC and retains the 30 most recent snapshots.

**Prerequisites:**
1. Add your Fly.io API token to GitHub Secrets:
   - Get token: `fly auth token` (run locally)
   - Go to your GitHub repo → Settings → Secrets and variables → Actions
   - Create a new secret named `FLY_API_TOKEN` with your token

2. **GitHub Issue Notifications:** When backups fail, the workflow automatically creates a GitHub Issue with the `backup-failure` label. You'll receive notifications if you're watching the repository. The issue includes a link to the failed run and details about which jobs failed.

**The workflow (`.github/workflows/backup.yml`) will:**
- Create daily snapshots for both `wiki_data` and `search_index` volumes
- Automatically delete snapshots older than 30 days
- Create a GitHub Issue on failure (prevents duplicate issues - adds comment to existing)
- Can be run manually via "Run workflow" button

**Manual snapshots** (before major changes):
```bash
# Create snapshot
fly volumes snapshots create wiki_data --app trevecca-pedia-wiki
fly volumes snapshots create search_index --app trevecca-pedia-search

# List snapshots
fly volumes snapshots list wiki_data --app trevecca-pedia-wiki
fly volumes snapshots list search_index --app trevecca-pedia-search
```

**View backup status:**
```bash
# Check recent snapshots
fly volumes snapshots list wiki_data --app trevecca-pedia-wiki
fly volumes snapshots list search_index --app trevecca-pedia-search

# View workflow runs
# Go to your GitHub repo → Actions → Daily Volume Backup
```

**Note:** This uses GitHub Actions free tier (2,000 minutes/month). Each backup run takes ~2-3 minutes, so well within free limits.

### Recovery Procedures

**If volume is lost, restore from snapshot:**

**Find the snapshot ID:**
```bash
# List available snapshots
fly volumes snapshots list wiki_data --app trevecca-pedia-wiki
fly volumes snapshots list search_index --app trevecca-pedia-search
```

**Restore from snapshot:**
```bash
# For wiki data
fly volumes create wiki_data --snapshot-id <snapshot-id> --region iad --size 1 --app trevecca-pedia-wiki

# For search index  
fly volumes create search_index --snapshot-id <snapshot-id> --region iad --size 1 --app trevecca-pedia-search
```

**Or let search service rebuild automatically:**
- Search service will automatically rebuild its index on startup
- No backup needed for search index volume

## Storage Usage Monitoring

Monitor volume usage:

```bash
# Check volume size and usage
fly volumes list --app trevecca-pedia-wiki
fly volumes list --app trevecca-pedia-search

# SSH and check actual usage
fly ssh console --app trevecca-pedia-wiki
du -sh /data/*

fly ssh console --app trevecca-pedia-search
du -sh /index
```

## Troubleshooting

### Volume not mounting
- Ensure volume is in the same region as the app
- Check that volume name in fly.toml matches created volume
- Verify the destination path exists in the Dockerfile

### Search index not building
- Check API_LAYER_URL is set correctly
- Check wiki service is healthy and accessible
- View logs: `fly logs --app trevecca-pedia-search`

### Database connection issues
- Verify WIKI_DB_HOST uses `.internal` suffix for fly.io networking
- Check firewall rules (fly.io internal networking is automatic)
- Verify database secrets are set correctly

## Free Tier Limits

**Current Usage:**
- **wiki_data:** 1GB volume
- **search_index:** 1GB volume  
- **Postgres:** Shared CPU, 1GB storage
- **Total:** ~3GB storage (within free tier limits)

**Upgrade if needed:**
- Increase volume size: `fly volumes extend <vol-id> --size 2`
- Dedicated CPU for better performance
- Multiple regions for redundancy

## Environment Variables Summary

### Wiki Service
- `DATABASE_URL` - Database url (secret)
- `WIKI_SERVICE_PORT` - Service port (9454)
- `WIKI_DATA_DIR` - Data directory (/data)

### Auth Service
- `DATABASE_URL` - Database url (secret)
- `PORT` - Service port (8083)
- `JWT_SECRET` - JWT signing key (secret; must match api-layer)
- `JWT_EXP_HOURS` - Token expiration (hours)
- `CORS_ORIGINS` - Allowed origins (optional; secret)
- `DEV_SEED` - Dev user seeding (false in prod)

### Search Service
- `API_LAYER_URL` - API layer URL (secret)
- `INDEX_DIR` - Index directory (/index)

### API Layer
- `WIKI_SERVICE_URL` - Wiki service URL (secret)
- `SEARCH_SERVICE_URL` - Search service URL (secret)
- `AUTH_SERVICE_URL` - Auth service URL (secret)
- `JWT_SECRET` - JWT verification key (secret; must match auth service)
- `API_LAYER_PORT` - Service port (2745)
