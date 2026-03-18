# Moderation Service

## Usage

Set up go:
```
go get moderation/cmd
```

Make sure to set up environment variables (in `moderation` directory):
```
cp .env.example ./.env
source .env
```

Using air in the `moderation` directory:
```
air .
```

## Info

This service starts an HTTP server on port `:7725`

## Endpoints

- `/health` - health check endpoint

For more info, check the [API Docs](../docs/api/moderation.md).
