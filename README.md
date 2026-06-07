# History API

A comprehensive RESTful API for Ultimate History Website, built with Go and the Fiber framework.

## Features

- **User Authentication**: Secure authentication with Google OAuth2 and JWT tokens
- **Role-based Access Control**: Granular permissions for different user roles
- **Geographic Data Management**: Handle historical geographic entities and spatial data
- **Media Management**: Upload, store, and manage historical media assets (images, documents)
- **Project Organization**: Group historical data into projects with member collaboration
- **Version Control**: Track changes and revisions to historical records
- **AI Chatbot**: Intelligent conversational interface for querying historical data
- **RAG System**: Retrieval-Augmented Generation for enhanced AI responses
- **Battle Replay**: Simulate and replay historical scenarios
- **Wiki System**: Collaborative documentation for historical topics
- **Statistics & Analytics**: Track usage and historical data metrics
- **Tile Services**: Serve map tiles for geographic visualization
- **Swagger Documentation**: Interactive API documentation

## Technology Stack

- **Language**: Go 1.26.3
- **Web Framework**: Fiber v3
- **Database**: PostgreSQL with PostGIS extension
- **Caching**: Redis
- **Object Storage**: AWS S3 compatible storage
- **Authentication**: JWT, Google OAuth2
- **API Documentation**: Swagger/OpenAPI
- **AI/ML**: LangChain Go for RAG and chatbot functionality
- **Geospatial**: pgvector for similarity search, MBTiles for map storage
- **Email**: SMTP integration for notifications
- **Validation**: Go Playground validator

## Project Structure

```
history-api/
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   └── worker/            # Background workers (cron jobs)
├── internal/              # Private application code
│   ├── controllers/       # HTTP request handlers
│   ├── dtos/              # Data Transfer Objects
│   ├── middlewares/       # Custom HTTP middlewares
│   ├── models/            # Data models and database entities
│   ├── repositories/      # Data access layer
│   ├── routes/            # Route definitions
│   └── services/          # Business logic
├── pkg/                   # Reusable packages
│   ├── ai/                # AI/RAG functionality
│   ├── cache/             # Redis caching
│   ├── config/            # Configuration management
│   ├── database/          # Database connections and seeding
│   ├── email/             # Email service
│   ├── log/               # Logging configuration
│   ├── mbtiles/           # Map tile storage
│   ├── oauth/             # OAuth providers
│   ├── constants/         # Application constants
│   └── storage/           # Abstract storage interface
├── assets/                # Embedded static assets
├── data/                  # Data files (MBTiles, etc.)
├── docs/                  # Documentation files
├── deployment/            # Deployment configurations
└── build/                 # Build artifacts
```

## Getting Started

### Prerequisites

- Go 1.26.3 or higher
- PostgreSQL 18+ with PostGIS extension
- Redis 6+
- AWS S3 account or compatible storage (MinIO, etc.)
- Google Cloud Project for OAuth2 authentication

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/history-api.git
   cd history-api
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Configure environment variables:
   Create a `.env` file in the `assets/resources` directory:
   ```env
    PROJECT_NAME=
    PGX_CONNECTION_URI=
    REDIS_CONNECTION_URI=
    POSTGRES_DB=
    POSTGRES_USER=
    POSTGRES_PASSWORD=
    SERVER_PORT=
    SERVER_IP=0.0.0.0

    JWT_SECRET=
    JWT_REFRESH_SECRET=

    SMTP_PASS=
    SMTP_USER=

    GOOGLE_CLIENT_ID=
    GOOGLE_CLIENT_SECRET=
    GOOGLE_REDIRECT_URL=

    STORAGE_ENDPOINT=
    STORAGE_ACCESS_KEY=
    STORAGE_SECRET_KEY=
    STORAGE_BUCKET_NAME=
    STORAGE_BUCKET_TEMP_NAME=
    STORAGE_REGION=


    ADMIN_DISPLAY_NAME=
    ADMIN_EMAIL=
    ADMIN_PASSWORD=

    EMAIL_WORKER_COUNT=2
    STORAGE_WORKER_COUNT=2
    RAG_WORKER_COUNT=1

    FRONTEND_URL=

    GOOGLE_AI_API_KEY
    GOOGLE_AI_MODEL=
    GOOGLE_AI_EMBEDDING_MODEL=

    OPEN_ROUTER_API=
    OPEN_ROUTER_MODEL=
    OPEN_ROUTER_EMBEDDING_MODEL=

    GOONG_API_KEY_MAP=
    GOONG_API_KEY_REQ=

    FIBER_BODY_LIMIT=0
    FIBER_CONCURRENCY=0
    FIBER_READ_BUFFER_SIZE=8192
    FIBER_WRITE_BUFFER_SIZE=8192
    FIBER_REDUCE_MEMORY_USAGE=false

    DISABLE_SWAGGER=false
    DISABLE_REQUEST_LOG=true
    DISABLE_STARTUP_MESSAGE=true
    ENABLE_PREFORK=false   

    PGX_MAX_CONNS=32
    PGX_MIN_CONNS=16
    PGX_MAX_CONN_IDLE_SECONDS=600
    PGX_HEALTH_CHECK_SECONDS=120

    REDIS_POOL_SIZE=64
    REDIS_MIN_IDLE_CONNS=16

    MBTILES_MAX_OPEN_CONNS=16
    MBTILES_MAX_IDLE_CONNS=8
   ```

4. Run the application:
   ```bash
    make dev
   ```
   
   Or build:
   ```bash
   make build
   ```

   Or build and run with docker:
   ```bash
   make docker-up
   ```

### API Documentation

Once the server is running, visit:
- Swagger UI: `http://localhost:${SERVER_PORT}/swagger/index.html`
- API endpoints: `http://localhost:${SERVER_PORT}/`

### Database Migrations
The application uses automatic seeding on startup. For production deployments, consider using a migration tool like:
- golang-migrate
- sqlc

## Configuration

All configuration is managed through environment variables or a `.env` file. See the [Configuration](#configuration) section above for details.

Key configuration areas:
- **Server**: Host, port, performance settings
- **Database**: Connection pooling, SSL, timezone
- **Cache**: Redis connection and TTL settings
- **Storage**: AWS credentials, bucket names, regions
- **Authentication**: JWT secrets, expiration, OAuth providers
- **AI Services**: LLM provider selection, API keys, model parameters
- **Email**: SMTP settings for notifications
- **Feature Flags**: Enable/disable specific modules


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Go Fiber](https://gofiber.io/) - Express inspired web framework
- [PostGIS](https://postgis.net/) - Spatial database extender for PostgreSQL
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search for PostgreSQL
- [LangChain Go](https://github.com/tmc/langchaingo) - Framework for LLM applications
- [Swagger UI](https://swagger.io/tools/swagger-ui/) - Interactive API documentation
- [MinIO](https://min.io/) - High performance S3 compatible object storage