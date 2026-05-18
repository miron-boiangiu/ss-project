# Role-Based Access Control (RBAC)

## Overview

RBAC is enforced on the Go backend using the `role` claim embedded in JWTs at login. The `withAuth` middleware extracts the role from the token into the request context, and individual handlers check authorization via `requireRole()`.

## Roles

| Role | Description |
|---|---|
| `user` | Default role assigned to all new registrations. Read-only access to photos. |
| `admin` | Elevated role set only via direct database manipulation (no API endpoint). Full access to all endpoints. |

## Permission Matrix

| Endpoint | Method | Description | `user` | `admin` |
|---|---|---|---|---|
| `/register` | POST | Create account | public | public |
| `/login` | POST | Authenticate | public | public |
| `/profile` | GET | Current user profile | ✅ | ✅ |
| `/photos` | GET | List/search photos | ✅ | ✅ |
| `/photos/{id}` | DELETE | Delete one photo | ❌ 401 | ✅ |
| `/photos/all` | DELETE | Delete all photos | ❌ 401 | ✅ |
| `/devices` | GET | List devices | ❌ 401 | ✅ |
| `/devices/switch` | POST | Switch device mode | ❌ 401 | ✅ |
| `/devices/command` | POST | Send command to device | ❌ 401 | ✅ |
| `/broker-info` | GET | MQTT broker connection info | public | public |

## Implementation Details

### JWT Claims (server/routes/user.go:99-103)

A JWT is issued at login containing:

```go
claims := jwt.MapClaims{
    "email": user.Email,
    "role":  user.Role,       // "user" or "admin"
    "exp":   time.Now().Add(time.Hour * 24).Unix(),
}
```

### Middleware (server/routes/init.go)

- `withAuth` — validates the JWT, extracts `email` and `role`, and injects them into `r.Context()`.
- `requireRole(r *http.Request, roles ...string) bool` — reads the `role` from context and returns `true` if it matches any of the allowed roles.

Usage in a handler:

```go
if !requireRole(r, roleAdmin) {
    http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
    return
}
```

### Role Constants (server/routes/init.go)

```go
const (
    roleUser  = "user"
    roleAdmin = "admin"
)
```

### Default Role on Registration (server/repository/user_repository.go:24)

All newly registered users are assigned the `"user"` role:

```go
Role: "user",
```

## How to Promote a User to Admin

Since there is no promotion API endpoint, insert/update the role directly in MongoDB:

```js
db.users.updateOne(
    { email: "user@example.com" },
    { $set: { role: "admin" } }
)
```

## Frontend

The client decodes the JWT payload to determine `isAdmin` (`client/src/contexts/AuthContext.tsx:36`). This is used for UI toggling (e.g. showing delete buttons) but **must not be relied upon for security** — all authorization is enforced server-side.

## Extending

To add a new role:

1. Add a constant in `server/routes/init.go`.
2. Assign the role to users (via registration logic or DB migration).
3. Call `requireRole(r, yourNewRole)` in the desired handlers.

To add fine-grained permissions, replace the `requireRole` role-string check with a permission-set lookup (e.g. `"photos:delete"`, `"devices:read"`) stored per role.
