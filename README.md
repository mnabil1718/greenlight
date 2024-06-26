# Migration

to avoid SSL error on postgres like:

```
error: pq: SSL is not enabled on the server
```

Disable the SSL mode with command below when migrating

```
migrate -path ./migrations -database "$GREENLIGHT_DB_DSN?sslmode=false" up
```
