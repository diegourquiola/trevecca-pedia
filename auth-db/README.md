# Auth Database Service

## Local Development

From `auth-db/`:

```bash
cp .env.example .env
docker compose up -d --force-recreate
```

Connect (example):

```bash
psql "host=localhost port=5433 dbname=auth user=auth_user password=$AUTH_DB_PASSWORD"
```

## Fly.io (Postgres)

This follows the same pattern as `wiki/` + `wiki-db/`.

1) Attach the Fly Postgres app to the auth service (this creates the DB + sets `DATABASE_URL` on the auth app):

```bash
fly postgres attach --app trevecca-pedia-auth --postgres-app trevecca-pedia-db
```

2) Apply schema to the created database:

```bash
./setup-db.sh trevecca-pedia-db trevecca_pedia_auth
```

Notes:
- DB names typically use underscores (e.g. `trevecca_pedia_auth`).
- Schema files live in `auth-db/init/`.
