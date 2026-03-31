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
* **`cmd/`**: Contains main entry points (e.g., `server/cmd/api/main.go`).
* **`internal/`**: Core logic isolated within bounded contexts.
  * **`controllers/` & `handlers/`**: Define HTTP handlers using Gin contexts.
  * **`routes/`**: Route definitions attaching handlers to Gin engines.
  * **`services/`**: Core business logic. Separates HTTP transport layer from database interactions.
  * **`repositories/`**: Explicit raw SQL database interaction logic.
  * **`models/`**: Domain structs and types.
  * **`errors/`**: Centralized typed errors.

## Style & Best Practices
* **Error Handling**: ELIMINATE string-based error handling. Strictly utilize typed, sentinel errors defined in `internal/errors/`.
* **Context Extraction Safety**: NEVER extract JWT variables using `c.MustGet(key)` directly, as token middleware slips or mock setups will yield hard server panics. ALWAYS use safe map handlers like `val, exists := c.Get(key)` yielding explicit `401 Unauthorized` responses before routing.
* **Database Parameter Validation**: Enforce strict `uuid.Validate(param)` checks natively across the Controller layer for all endpoints retrieving UUIDs. Prevent 500-level database crashes from invalid inputs before any `pgx` executions.
* **Concurrency**: Leverage Go's lightning-fast concurrency model. Design the API to elegantly handle hundreds of simultaneous Server-Sent Events (SSE) connections for live leaderboards without choking memory.
* **Raw SQL**: Write efficient raw PostgreSQL queries via `pgx/v5`. Do not bring in ORMs. Keep queries optimized and indexed.

## Developer Workflow
* **Start Dev Server**: `air` (Uses `.air.toml` for Go live reloading)
