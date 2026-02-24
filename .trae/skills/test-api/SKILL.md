---
name: test api
description: test backend api
---

use following workflow:

1. use dev-manager mcp to start the backend server, command to start backend dev server (must run in backend/ directory): `MARKPOST_SERVER__PORT=$PORT go run .`
2. if need authorization, use this account to login. username: markpost, password: markpost. if authorization failed, ask use to provide a correct account.
3. use curl to test the api
