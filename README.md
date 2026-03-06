# CRM Data Platform

A multi-tenant CRM data platform built with Go, PostgreSQL, and Elasticsearch. Designed to scale to billions of records with tenant isolation via Row Level Security.

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│   Gin HTTP   │────▶│  CRM Service │────▶│   PostgreSQL 16  │
│   Router     │     │  (App Layer) │     │  (Source of Truth)│
│              │     │              │────▶│  + RLS Policies   │
│  Middleware: │     │  - CRUD      │     └──────────────────┘
│  - Tenant    │     │  - Search    │
│  - Error     │     │  - Lifecycle │     ┌──────────────────┐
└──────────────┘     │  - Assoc     │────▶│  Elasticsearch 8 │
                     └──────────────┘     │  (Search/Filter)  │
                                          └──────────────────┘
```

**Layered Architecture **
- `cmd/` - Entry point, dependency injection
- `config/` - Environment-based configuration
- `internal/domain/` - Entities, value objects, repository interfaces
- `internal/application/` - Business logic, DTOs, controllers
- `internal/infrastructure/` - Database, Elasticsearch, HTTP, middleware

## Architectural Patterns & Tech Stack

This project follows my established architectural patterns for Go-based microservices, utilizing a **Domain-Driven Design (DDD)** structure. This ensures a clear separation of concerns between the business logic (Domain), the implementation (Infrastructure), and the API (Application), which is critical for long-term maintainability in a platform of this scale.

- **Framework:** [Gin](https://gin-gonic.com/) for high-performance HTTP routing and middleware.
- **ORM:** [GORM](https://gorm.io/) for developer-friendly database interactions and schema management.
- **Service Layer:** Implements a generic object pattern to handle both core CRM entities and custom objects through a unified service interface.
- **Search Engine:** Integrated with **Elasticsearch 8** for distributed search and filtering, utilizing shard routing for tenant isolation.

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Multi-tenancy | Shared schema + RLS | Single migration set, shared pool, DB-enforced isolation |
| Custom fields | JSONB column | 3x smaller than EAV, single-row reads, GIN indexable |
| Relationships | Generic associations table | Supports any-to-any links between standard and custom objects |
| Search | Elasticsearch with routing | Shard-level tenant isolation, sub-50ms P95 for filtered queries |
| API pattern | Generic /objects/{type} | One controller handles all object types, mirrors HubSpot pattern |

## Core Entities

- **Contacts** - People with email, phone, tags, custom fields
- **Companies** - Organizations with domain, industry, revenue
- **Opportunities** - Deals linked to pipelines, stages, contacts, companies
- **Pipelines** - Sales pipelines with ordered stages
- **Custom Objects** - Tenant-defined entities (e.g., Policies, Vehicles, Properties)

## Multi-Tenant Isolation

Every request requires `X-Tenant-Id` header. The middleware extracts it and the database layer executes `SET LOCAL app.current_tenant_id` before every query. PostgreSQL RLS policies enforce that queries only see/modify data for the current tenant. This is defense in depth: even application bugs cannot leak data across tenants.

## API Endpoints

```
POST   /crm/objects/:type              Create record
GET    /crm/objects/:type              List records
GET    /crm/objects/:type/:id          Get record
PATCH  /crm/objects/:type/:id          Update record
DELETE /crm/objects/:type/:id          Soft-delete record
POST   /crm/objects/:type/search       Search/filter records
PATCH  /crm/objects/:type/:id/archive  Archive record
PATCH  /crm/objects/:type/:id/restore  Restore record

POST   /crm/schemas                    Create custom object schema
GET    /crm/schemas                    List schemas
POST   /crm/pipelines                  Create pipeline with stages
POST   /crm/association-definitions    Define relationship type
```

## Project Structure

```
├── cmd/main.go                    Entry point
├── config/config.go               Configuration
├── internal/
│   ├── application/crm/
│   │   ├── controller.go          HTTP handlers
│   │   ├── service.go             Business logic
│   │   ├── dto.go                 Request/response types
│   │   └── mapper.go              Entity <-> DTO mapping
│   ├── domain/
│   │   ├── entity/                Contact, Company, Opportunity, Pipeline, CustomObject, Association
│   │   ├── repository/            Interfaces for data access
│   │   └── valueobject/           LifecycleState, FieldType, Cardinality, Filter
│   └── infrastructure/
│       ├── database/              GORM repositories, tenant DB, scopes
│       ├── elasticsearch/         ES client, search repo, sync service
│       ├── http/                  Gin router
│       └── middleware/            Tenant extraction, error handling
├── elasticsearch/mappings/        ES index mappings (contacts, companies, opportunities, custom_objects)
├── docs/
│   ├── data-model.md              ERD, DDL (SQL snippets), multi-tenant strategy
│   ├── storage-indexing.md        Storage engines, indexing, consistency
│   ├── api-contracts.md           Full API reference, filter DSL, versioning
│   └── reliability-safety.md      Reliability, isolation, and scale essay
```

## Trade-Offs

**Shared schema + RLS vs schema-per-tenant**: Chose shared for operational simplicity. Trade-off is that one poorly-indexed query can impact the shared pool. Mitigated by per-tenant query timeouts and monitoring.

**JSONB vs EAV for custom fields**: Chose JSONB for performance and simplicity. Trade-off is that schema validation lives in application code rather than DB constraints. Mitigated by CustomObjectSchema field definitions and application-level validation.

**Synchronous ES sync vs CDC**: Chose synchronous dual-write for simplicity. Trade-off is added latency on writes (~5-10ms) and potential inconsistency if ES write fails (logged, not retried). Path to CDC via Debezium documented in Part 2.

**Offset pagination vs cursor**: Chose offset for simplicity. Trade-off is performance degrades at high offsets. Path to cursor-based pagination documented in Part 3.

