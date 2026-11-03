# Quickstart

```bash
make up
make verify
```

`make up` prints the initial administrator password exactly when it creates `secrets/admin_password.txt`.
Use `AUTH_API_PORT` / `AUTH_API_MANAGEMENT_PORT` to avoid host port conflicts; `make verify` uses the same ports.

The default administrator username is `admin`. The API exposes:

- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `GET /api/v1/users`
- `GET /api/v1/roles`
- `GET /api/v1/permissions`

Stop the stack with `make down`; remove volumes as well with `make clean`.
