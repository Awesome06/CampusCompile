# Server (Backend API) Agent Rules

## Tech Stack & Key Libraries
* **Language**: Go 1.25.0
* **Routing / Web Framework**: `gin-gonic/gin`
* **Database Driver**: `jackc/pgx/v5` (Native PostgreSQL driver, NO ORMs like GORM are used or permitted)
* **Message Broker / Caching**: `redis/go-redis/v9`
* **Cloud / Storage**: `aws-sdk-go-v2` (S3, STS, SSO interactions)
* **Auth**: `golang-jwt/jwt/v5`, Azure AD via `golang.org/x/oauth2`
* **UUIDs**: `google/uuid`

## Architecture & Conventions
* **`cmd/`**: Contains main entry points (e.g., `cmd/server/main.go`).
* **`internal/`**: Core logic isolated within bounded contexts.
  * **`controllers/` & `handlers/`**: Define HTTP handlers using Gin contexts.
  * **`routes/`**: Route definitions attaching handlers to Gin engines.
  * **`services/`**: Core business logic. Separates HTTP transport layer from database interactions.
  * **`repositories/`**: Explicit raw SQL database interaction logic.
  * **`models/`**: Domain structs and types.
  * **`errors/`**: Centralized typed errors.

## Style & Best Practices
* **Error Handling**: ELIMINATE string-based error handling. Strictly utilize typed, sentinel errors defined in `internal/errors/`. Prevent 500-level crashes from invalid inputs (e.g. failing to cast/parse inputs blindly).
* **Validation**: Enforce strict UUID validation for all incoming database requests at the controller/handler level.
* **Concurrency**: Leverage Go's lightning-fast concurrency model. Design the API to elegantly handle hundreds of simultaneous Server-Sent Events (SSE) connections for live leaderboards without choking memory.
* **Raw SQL**: Write efficient raw PostgreSQL queries via `pgx/v5`. Do not bring in ORMs. Keep queries optimized and indexed.

## Developer Workflow
* **Start Dev Server**: `air` (Uses `.air.toml` for Go live reloading)
