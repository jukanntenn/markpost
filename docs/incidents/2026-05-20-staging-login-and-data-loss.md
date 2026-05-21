# Staging: Login Failure and Data Loss

**Date**: 2026-05-20
**Environment**: staging (markpost.bytehome.fun)

## Issues

### 1. Login form submits to localhost

**Symptom**: Submitting login at `https://markpost.bytehome.fun/login` sends requests to `http://localhost:7330/api/v1/auth/login`.

**Root cause**: `frontend/.env.local` sets `NEXT_PUBLIC_API_URL=http://localhost:7330`, which gets baked into the client bundle at build time. The Dockerfile `COPY . .` included `.env.local`, overriding the `ENV NEXT_PUBLIC_API_URL=` set in the Dockerfile (`.env.local` has higher precedence in Next.js).

**Fix**:
- Added `frontend/.dockerignore` to exclude `.env.local` from the build context.
- Set `ENV NEXT_PUBLIC_API_URL=` in `docker/frontend.Dockerfile` before `pnpm build`.
- Removed `src/middleware.ts` — Next.js 16 uses `src/proxy.ts` (already existed). Middleware file caused a build conflict.
- Client now uses relative URLs (`/api/v1/...`), and the Next.js proxy rewrites to `API_PROXY_TARGET`.

### 2. User password not working (password_hash is NULL)

**Symptom**: Login fails with "user has no password set" for the `markpost` user.

**Root cause**: Commit `2043d14` renamed the GORM column tag from `gorm:"not null"` (column: `password`) to `gorm:"column:password_hash"`. GORM's `AutoMigrate` created a new empty `password_hash` column but left the old `password` column intact with the data. The app reads from the new empty column.

**Fix**: Added `migratePasswordColumn()` in `backend/internal/infra/db.go` — copies data from `password` to `password_hash` when the legacy column exists.

### 3. All posts deleted after deploy

**Symptom**: After deploying the password migration, all 1349 posts disappeared.

**Root cause**: `migratePasswordColumn()` called `DropColumn(&user.User{}, "password")`. In SQLite, dropping a `NOT NULL` column requires a table rebuild (create new → copy data → **drop old table** → rename). When the old `users` table is dropped, the `OnDelete:CASCADE` constraint on `posts` and `delivery_channels` fires, deleting all dependent records.

**Fix**: Removed the `DropColumn` call. The legacy `password` column is left in place as an orphan — harmless and safer than risking another table rebuild.

## Files Changed

- `docker/frontend.Dockerfile` — added `ENV NEXT_PUBLIC_API_URL=`
- `frontend/.dockerignore` — exclude `.env.local`
- `frontend/src/middleware.ts` — deleted (conflicts with Next.js 16 proxy)
- `backend/internal/infra/db.go` — added `migratePasswordColumn()` (data copy only, no drop)
