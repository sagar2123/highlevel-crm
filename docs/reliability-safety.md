# Reliability, Isolation & Safety at Scale

## 1. Reliability & Performance Budgets

Performance budgets are not aspirational targets; they are hard constraints that shape every architectural decision. When a latency ceiling is defined before implementation begins, it eliminates entire categories of design choices and forces the system toward provably fast paths.

### CRUD Operations

| Percentile | Target   |
|------------|----------|
| P50        | < 10 ms  |
| P95        | < 50 ms  |
| P99        | < 200 ms |

These budgets are achievable because every CRUD path resolves to a single-row lookup by primary key or a narrow index scan. The critical enablers:

- **Composite indexes with `tenant_id` as leading column.** Every query the application issues is tenant-scoped. Placing `tenant_id` first in every composite index means the planner immediately narrows to a small slice of the B-tree, keeping index scans well under 10 ms for the common case.
- **Connection pooling and prepared statements.** GORM's built-in pool eliminates connection setup overhead. Prepared statements avoid repeated parse and plan cycles for hot queries.
- **JSONB over EAV for custom fields.** This is a budget-driven decision. Reading a contact with 50 custom fields under the EAV model requires a join that fans out to 50 rows, aggregates them, and returns. Under JSONB, it is a single row read with a single index lookup. The P50 target of 10 ms rules out EAV entirely.

### Query and Search Endpoints

| Percentile | Target   |
|------------|----------|
| P50        | < 50 ms  |
| P95        | < 200 ms |
| P99        | < 500 ms |

Search is where the architecture diverges from the source-of-truth database. Complex filter combinations, full-text search, and sorted pagination across millions of records cannot meet a 200 ms P95 in PostgreSQL without extreme index proliferation. Instead:

- **Elasticsearch with routing by `tenant_id`.** Routing ensures all documents for a tenant land on the same shard. A query that would otherwise fan out to every shard in the cluster now hits exactly one. This alone cuts search latency by a factor proportional to shard count.
- **Denormalized documents.** The contact document in ES embeds `company_name` directly rather than requiring a cross-index join at query time. The budget makes this tradeoff obvious: a join across ES indices would blow through the 200 ms ceiling for any tenant with meaningful data volume.
- **Protecting the source-of-truth database.** All complex filter and search traffic is routed to ES. PostgreSQL handles only direct CRUD and simple lookups. This separation prevents a surge of analytical queries from degrading write latency.

### Background Jobs

| Job Type       | Target Lag       | Constraint                                       |
|----------------|------------------|--------------------------------------------------|
| ES sync        | < 5 seconds      | Near-real-time; users expect search to reflect writes quickly |
| Analytics ETL  | < 5 minutes      | Acceptable for dashboards and reports             |
| Backfill jobs  | Best-effort      | Rate-limited to avoid impacting live traffic      |

Backfill jobs are throttled explicitly: batch updates with `WHERE id > last_processed_id LIMIT 10000`, paced at one batch per second. Without this throttle, a backfill across billions of rows would saturate disk I/O and evict hot pages from the buffer pool, violating every CRUD latency budget.

---

## 2. Multi-Tenant Isolation & Blast Radius

In a multi-tenant system, the fundamental risk is not just a security breach -- it is the operational reality that one tenant's behavior can degrade the experience for every other tenant sharing the same infrastructure.

### Database Layer Isolation

**Row-Level Security is the hard boundary.** RLS policies enforce `tenant_id = current_setting('app.current_tenant_id')` at the PostgreSQL level. Even if the application contains a bug -- a missing WHERE clause, an ORM misconfiguration, a malformed query -- the database itself refuses to return rows belonging to another tenant. This is defense in depth: the application sets the tenant context via `SET LOCAL` at transaction start, and the database enforces it regardless of what queries follow.

`SET LOCAL statement_timeout = '5s'` is applied per transaction. This prevents a single tenant's runaway query (an unindexed filter on a 100M row table) from holding a connection and blocking other tenants. The connection pool is shared, so a stuck connection is a shared resource consumed. The timeout ensures it is released.

### Per-Tenant Quotas

- **Rate limiting at the API gateway:** 100 requests per second per `tenant_id`. This is the first line of defense against both abuse and accidental tight loops in client integrations.
- **Record count limits per object type:** Configurable per tenant (default: 10M contacts). Prevents unbounded growth that would degrade index performance for co-located tenants.
- **Search query complexity limits:** Maximum filter depth, maximum `page_size`, maximum aggregation cardinality. These prevent a single complex query from consuming disproportionate ES resources.

### Noisy Neighbor Prevention

Quotas limit the blast radius of intentional load. Monitoring addresses the unintentional kind:

- **Per-tenant latency and throughput tracking.** When a specific `tenant_id` begins consuming an outsized share of database time, the system can identify it before other tenants are affected.
- **Dedicated read replicas for large tenants.** A tenant with more than 10M records generates fundamentally different query patterns. Routing their read traffic to a dedicated replica prevents their large sequential scans from evicting cached pages that smaller tenants depend on.
- **Elasticsearch shard isolation.** Because routing pins a tenant to specific shards, a hot tenant's data can be moved to dedicated hardware without re-indexing the entire cluster.
- **Circuit breaker on Elasticsearch.** If ES becomes unavailable, the system degrades gracefully: simple queries fall back to PostgreSQL, complex search returns an explicit service degradation error. The source-of-truth database is never exposed to the full search query load.

### Cell-Based Architecture (Future State)

The natural evolution of tenant isolation is the cell model. Each cell is a fully independent stack: its own PostgreSQL instance, its own ES cluster, its own application tier. Tenants are assigned to cells based on load. Large tenants receive dedicated cells. The blast radius of any infrastructure failure -- a bad deploy, a hardware fault, a corrupted index -- is limited to a single cell. This is the architecture that allows the platform to scale to tens of thousands of tenants without compounding operational risk.

---

## 3. Migrations & Evolution

A system with billions of records cannot afford downtime for schema changes. Every migration strategy must be evaluated against the question: does this block writes, and for how long?

### Schema Migrations at Scale

- **Column additions** are instant in PostgreSQL 11 and later when a DEFAULT is specified. The database writes the default into the catalog rather than rewriting every row. This is safe to run in production without coordination.
- **Index creation** uses `CREATE INDEX CONCURRENTLY`, which builds the index without holding a lock that blocks writes. For large tables, `pg_repack` handles cases where the table itself needs reorganization.
- **Column type changes** are never done in place on large tables. The safe pattern: add a new column with the target type, backfill in batches (same throttled approach as background jobs), update the application to read from the new column, drop the old column. Each step is independently reversible.

### Rolling Out New Indexing Strategies in Elasticsearch

The blue-green index pattern eliminates downtime for mapping changes:

1. Create a new index with the updated mappings.
2. Reindex from PostgreSQL (not from the old ES index, to avoid propagating stale data).
3. Validate: compare document counts, spot-check random records, run representative queries against both indices.
4. Swap the alias atomically from the old index to the new one.
5. Decommission the old index after a soak period.

At no point does the application need to be aware of the migration. It reads and writes through the alias, and the alias always points to a valid, fully populated index.

### Decomposing the Monolithic CRM Store

As the platform grows, different object types (contacts, opportunities, companies) will develop divergent access patterns and scaling requirements. The strangler fig pattern provides a safe extraction path:

1. **Extract one object type at a time** into its own service with its own data store.
2. **Dual-write phase:** Both the legacy store and the new service receive writes. This provides a rollback path -- if the new service has issues, the legacy store is still authoritative.
3. **Read migration:** Gradually shift read traffic to the new service behind a feature flag, starting at a small percentage and increasing as confidence grows.
4. **Reconciliation:** A background job continuously compares records between the old and new stores, flagging any divergence.
5. **Event-driven decoupling:** Introduce domain events (`ContactCreated`, `OpportunityStageChanged`) so that downstream consumers (search indexing, analytics, notifications) are decoupled from the source service. This prevents the extraction of one service from requiring coordinated changes across every consumer.

### Feature Flags & Staged Rollouts

No schema change or behavioral change is deployed to all tenants simultaneously:

- New schema-dependent features are gated behind feature flags, rolled out progressively: 1% of tenants, then 10%, then 50%, then 100%.
- Canary deployments target a single cell first. Error rates and latency are monitored against baseline before proceeding.
- Every migration has a corresponding down migration. Every feature flag can be disabled instantly. The ability to roll back within seconds is not optional -- it is a prerequisite for deploying changes to a system that other businesses depend on for their daily operations.

---

**The through-line across reliability, isolation, and migration strategy is the same principle: limit the blast radius.** Performance budgets limit the blast radius of bad design choices. Tenant isolation limits the blast radius of noisy neighbors. Cell architecture limits the blast radius of infrastructure failures. Staged rollouts limit the blast radius of bad deploys. At staff-level scale, the system's resilience is not defined by how well it handles the happy path, but by how tightly it constrains the damage when something goes wrong.
