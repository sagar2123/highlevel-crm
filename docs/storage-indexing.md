# Storage & Indexing Strategy Across Multiple Engines

## Overview

This document describes the storage and indexing architecture for the CRM Data Platform. The system uses a polyglot persistence approach where different storage engines serve different access patterns:

- **PostgreSQL** -- source of truth for all CRM data (contacts, companies, opportunities, custom objects)
- **Elasticsearch** -- derived read-optimized store for search, filtering, and full-text queries
- **ClickHouse / BigQuery** -- analytics layer for dashboards, aggregations, and historical reporting (discussed but not implemented in the current codebase)

All stores operate in a multi-tenant model with `tenant_id` as the tenant identifier. Tenant isolation is enforced at every layer: Row-Level Security policies in PostgreSQL, mandatory routing and term filters in Elasticsearch, and tenant-scoped partitions in the analytics tier.

---

## 1. Source of Truth vs Derived Stores

### PostgreSQL -- Source of Truth

PostgreSQL is the authoritative store for all CRM data. Every write operation (create, update, delete, archive) targets PostgreSQL first. It provides:

- **ACID transactions** -- every mutation is wrapped in a transaction with tenant context set via `SET LOCAL app.current_tenant_id`, ensuring atomicity and isolation.
- **Referential integrity** -- foreign keys enforce relationships between contacts and companies, opportunities and pipelines/stages, custom object records and schemas, and associations between any two record types.
- **Row-Level Security** -- every table has an RLS policy that filters rows by `tenant_id = current_setting('app.current_tenant_id')::uuid`, guaranteeing that tenant data is never leaked even if application code has a bug.
- **Schema enforcement** -- built-in types (contacts, companies, opportunities) have strongly typed columns, while extensibility is provided through JSONB `custom_properties` columns.

The PostgreSQL schema (from `migrations/000003_create_crm_tables.up.sql`) defines the canonical structure:

```sql
CREATE TABLE contacts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    first_name        VARCHAR(255),
    last_name         VARCHAR(255),
    email             VARCHAR(320),
    phone             VARCHAR(50),
    company_id        UUID REFERENCES companies(id),
    source            VARCHAR(100),
    tags              TEXT[],
    custom_properties JSONB DEFAULT '{}',
    lifecycle_state   lifecycle_state NOT NULL DEFAULT 'active',
    created_by        UUID,
    updated_by        UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);
```

### Elasticsearch -- Derived Search Store

Elasticsearch is a read-optimized derived store. It is never written to directly by clients; it receives data exclusively from the application service layer after a successful PostgreSQL write. It provides:

- **Full-text search** -- custom analyzers for names (asciifolding, lowercase) and emails (uax_url_email tokenizer) enable fuzzy, accent-insensitive search.
- **Faceted filtering** -- keyword sub-fields allow exact-match filtering and sorting on fields like `lifecycle_state`, `source`, `tags`, `pipeline_name`, and `stage_name`.
- **Denormalized documents** -- related entity names (company_name, contact_name, pipeline_name, stage_name) are embedded in ES documents to avoid cross-index joins.
- **Dynamic custom properties** -- the `custom_properties` (or `properties` for custom objects) field uses `"dynamic": true` so tenant-defined fields are automatically indexed.

If Elasticsearch is temporarily unavailable, the application continues to serve reads from PostgreSQL (list endpoints hit PG directly). Search functionality degrades gracefully.

### Analytics -- ClickHouse / BigQuery (Future)

The analytics tier is designed for workloads that are poor fits for both PostgreSQL and Elasticsearch:

- **Dashboard aggregations** -- "total pipeline value by stage this quarter" across millions of opportunities.
- **Historical reporting** -- time-series analysis of contact acquisition, deal velocity, conversion funnels.
- **Cross-tenant platform analytics** -- aggregate metrics for the platform operator (not exposed to tenants).

Acceptable latency for analytics data is 1-5 minutes behind the source of truth, loaded via CDC or batch ETL.

### Clear Ownership: Write Path

```
Client Request
    |
    v
Application Service (service.go)
    |
    |---> PostgreSQL WRITE (source of truth, in transaction)
    |         |
    |         |--- success ---+
    |                         |
    |                         v
    |                   Elasticsearch INDEX (fire-and-forget, log on failure)
    |
    v
Response to Client
```

All writes go to PostgreSQL first. The ES index operation happens after the PG transaction commits. If ES indexing fails, the data is still safe in PostgreSQL and the ES document will be reconciled by a drift detection job (described in Section 4).

---

## 2. Indexing & Denormalization for Elasticsearch

### Contacts Index (`contacts`)

The contacts index stores a denormalized representation of each contact:

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "loc-uuid-here",
  "first_name": "Maria",
  "last_name": "Garcia",
  "full_name": "Maria Garcia",
  "email": "maria.garcia@example.com",
  "phone": "+1-555-0142",
  "company_id": "comp-uuid-here",
  "company_name": "Acme Corp",
  "source": "web_form",
  "tags": ["enterprise", "inbound"],
  "lifecycle_state": "active",
  "custom_properties": {
    "lead_score": 85,
    "preferred_language": "es",
    "utm_source": "google_ads"
  },
  "created_at": "2025-11-15T08:30:00Z",
  "updated_at": "2025-12-01T14:22:00Z"
}
```

Key denormalization decisions:

- `full_name` is a computed field (`first_name + " " + last_name`) that enables single-field name search without requiring a `multi_match` across two fields.
- `company_name` is denormalized from the related `companies` table so that filtering contacts by company name does not require a join or secondary lookup.
- `custom_properties` is a dynamic object -- when a tenant adds a custom field like `lead_score`, Elasticsearch automatically creates a mapping for it.

### Opportunities Index (`opportunities`)

```json
{
  "id": "opp-uuid-here",
  "tenant_id": "loc-uuid-here",
  "name": "Acme Corp - Enterprise License",
  "pipeline_id": "pipe-uuid-here",
  "pipeline_name": "Sales Pipeline",
  "stage_id": "stage-uuid-here",
  "stage_name": "Negotiation",
  "contact_id": "contact-uuid-here",
  "contact_name": "Maria Garcia",
  "company_id": "comp-uuid-here",
  "company_name": "Acme Corp",
  "monetary_value": 50000,
  "currency": "USD",
  "expected_close_date": "2026-03-15T00:00:00Z",
  "assigned_to": "user-uuid-here",
  "lifecycle_state": "active",
  "custom_properties": {
    "deal_source": "referral",
    "competitor": "SalesCo"
  },
  "created_at": "2025-10-20T09:00:00Z",
  "updated_at": "2025-12-10T16:45:00Z"
}
```

Denormalized fields: `pipeline_name`, `stage_name`, `contact_name`, and `company_name` are resolved at index time. When a pipeline stage is renamed, a background job must re-index all opportunities in that pipeline to keep ES consistent (or the rename operation can fan out updates).

### Custom Objects Index (`custom_objects`)

Custom objects use a shared index with discriminator fields:

```json
{
  "id": "rec-uuid-here",
  "tenant_id": "loc-uuid-here",
  "schema_id": "schema-uuid-here",
  "object_type": "vehicles",
  "properties": {
    "make": "Tesla",
    "model": "Model 3",
    "year": 2025,
    "vin": "5YJ3E1EA1NF000001"
  },
  "lifecycle_state": "active",
  "created_at": "2025-11-01T12:00:00Z",
  "updated_at": "2025-11-15T08:00:00Z"
}
```

A shared index is used instead of per-schema indexes because:

- Tenants can create an unbounded number of custom object types -- one index per type would cause shard explosion.
- The `schema_id` and `object_type` keyword fields allow efficient filtering within the shared index.
- Dynamic mapping on the `properties` object lets each schema define its own fields without upfront mapping configuration.

### Tenant Boundaries: Routing by `tenant_id`

Every ES index is configured with `"_routing": { "required": true }`. All index and search operations pass `tenant_id` as the routing value:

```go
// Indexing (from search_repo.go)
r.client.Index(
    objectType,
    &buf,
    r.client.Index.WithDocumentID(id),
    r.client.Index.WithRouting(tenantID),
)

// Searching (from search_repo.go)
r.client.Search(
    r.client.Search.WithIndex(objectType),
    r.client.Search.WithBody(&buf),
    r.client.Search.WithRouting(tenantID),
)
```

This guarantees:

1. All documents for a given tenant land on the same shard (deterministic routing by `tenant_id`).
2. Search queries skip N-1 shards, hitting only the shard that contains the tenant's data.
3. Combined with the mandatory `term` filter on `tenant_id` in every query, there is no possibility of cross-tenant data leakage.

---

## 3. Example Search Index Schema

The full mapping for the contacts index (`elasticsearch/mappings/contacts.json`):

```json
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "email_analyzer": {
          "type": "custom",
          "tokenizer": "uax_url_email",
          "filter": ["lowercase"]
        },
        "name_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding"]
        }
      }
    }
  },
  "mappings": {
    "_routing": { "required": true },
    "properties": {
      "id":                { "type": "keyword" },
      "tenant_id":       { "type": "keyword" },
      "first_name":        {
        "type": "text",
        "analyzer": "name_analyzer",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "last_name":         {
        "type": "text",
        "analyzer": "name_analyzer",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "full_name":         { "type": "text", "analyzer": "name_analyzer" },
      "email":             {
        "type": "text",
        "analyzer": "email_analyzer",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "phone":             { "type": "keyword" },
      "company_id":        { "type": "keyword" },
      "company_name":      {
        "type": "text",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "source":            { "type": "keyword" },
      "tags":              { "type": "keyword" },
      "lifecycle_state":   { "type": "keyword" },
      "custom_properties": { "type": "object", "dynamic": true },
      "created_at":        { "type": "date" },
      "updated_at":        { "type": "date" }
    }
  }
}
```

### Analyzer Design Rationale

| Analyzer | Tokenizer | Filters | Purpose |
|---|---|---|---|
| `email_analyzer` | `uax_url_email` | `lowercase` | Treats email addresses as single tokens. Searching "maria" matches "maria.garcia@example.com" via the text field, while `email.keyword` supports exact-match lookups. |
| `name_analyzer` | `standard` | `lowercase`, `asciifolding` | Handles international names. "Garcia" matches "garcia", "Garci'a" matches "garcia", and "Munoz" matches "munoz" (asciifolding normalizes diacritics). |

### Field Design Patterns

- **Text + Keyword sub-field** (`first_name`, `last_name`, `email`, `company_name`): The `text` type enables full-text search with the appropriate analyzer. The `.keyword` sub-field enables exact-match filtering (`term` queries) and sorting. Example: sort by `last_name.keyword` ascending.
- **Keyword-only fields** (`id`, `tenant_id`, `source`, `tags`, `lifecycle_state`, `phone`): These fields are used exclusively for exact-match filtering and aggregations. No tokenization needed.
- **Date fields** (`created_at`, `updated_at`): Enable range queries for temporal filtering (e.g., "contacts created in the last 30 days") and date histogram aggregations.
- **Dynamic object** (`custom_properties`): New fields are automatically detected and mapped by Elasticsearch. String values become `text` with a `.keyword` sub-field; numbers become `long` or `float`; booleans become `boolean`.

---

## 4. Consistency & Latency Trade-Offs

The system operates with three consistency tiers:

### Tier 1: Strong Consistency -- PostgreSQL

All CRUD operations hit PostgreSQL directly. Read-after-write consistency is guaranteed within the same request. The `ListRecords` and `GetRecord` endpoints read from PostgreSQL, so a client that creates a contact and immediately lists contacts will always see their new record.

```
Write  -->  PostgreSQL  -->  Read (same request or subsequent)
            (ACID)          Result: guaranteed to see the write
```

### Tier 2: Eventual Consistency -- Elasticsearch (2-5 seconds)

The current implementation uses synchronous dual-write within the application service. After a successful PG commit, the service issues an ES index request in the same HTTP request handler:

```go
// From service.go -- CreateRecord for contacts
if err := s.contacts.Create(ctx, contact); err != nil {
    return nil, err
}
resp := toRecordResponse(objectType, contact.ID.String(), ...)
s.sync.IndexDocument(ctx, objectType, contact.ID.String(), resp.Properties)
return resp, nil
```

The ES write is fire-and-forget: if it fails, the error is logged but the HTTP response still succeeds with the PG data. This means:

- **Happy path**: ES is updated within the same request (~50-200ms additional latency). ES `refresh_interval` (default 1 second) means the document becomes searchable within 1-2 seconds.
- **Failure path**: ES index fails, the document is missing from search results until the next update or the drift reconciliation job runs.
- **Stale read window**: Between the PG commit and ES refresh, a search query may not return the newly created record. This is acceptable because the `ListRecords` endpoint (which is the primary "list all" endpoint) reads from PG.

### Tier 3: Acceptable Lag -- Analytics (1-5 minutes)

The analytics tier (ClickHouse/BigQuery) operates on a separate pipeline:

- **CDC approach**: Debezium reads the PostgreSQL WAL and publishes change events to Kafka. A consumer writes to the analytics store.
- **Batch ETL approach**: Periodic queries against PostgreSQL extract changed records (using `updated_at` watermarks) and bulk-load them.
- **Acceptable lag**: 1-5 minutes. Dashboard queries are inherently tolerant of slight staleness.

### Drift Detection and Reconciliation

Even with synchronous dual-write, drift between PostgreSQL and Elasticsearch can occur due to:

- ES indexing failures (network issues, cluster overload)
- Partial failures in bulk operations
- Direct database modifications (migrations, data fixes)

The reconciliation strategy:

1. **Periodic count comparison**: A scheduled job queries `SELECT count(*), tenant_id FROM contacts WHERE lifecycle_state = 'active' GROUP BY tenant_id` and compares with ES `_count` queries per tenant.

2. **Checksum comparison**: For tenants with count mismatches, compute a checksum over `(id, updated_at)` tuples in PG and compare with ES. This identifies which specific records have drifted.

3. **Targeted re-index**: Only the drifted records are re-indexed from PG to ES.

4. **Full re-index**: If drift exceeds a threshold (e.g., >5% of records), trigger a full re-index for that tenant using the bulk API.

All sync operations are idempotent. ES documents are indexed by their PG `id`, so re-indexing the same record simply overwrites the existing document with the current PG state.

---

## 5. Indexing Pipeline

### Current Implementation: Synchronous Dual-Write

The current codebase implements synchronous dual-write in the application service layer (`internal/application/crm/service.go`). The `SyncService` (`internal/infrastructure/elasticsearch/sync.go`) wraps the ES repository:

```go
type SyncService struct {
    search repository.SearchRepository
}

func (s *SyncService) IndexDocument(ctx context.Context, objectType string, id string, doc map[string]interface{}) {
    if err := s.search.Index(ctx, objectType, id, doc); err != nil {
        log.Printf("failed to index %s/%s: %v", objectType, id, err)
    }
}
```

The write path for each operation:

| Operation | PG Action | ES Action |
|---|---|---|
| Create | `INSERT` | `Index` (create document) |
| Update | `UPDATE` | `Index` (overwrite document) |
| Delete | `UPDATE lifecycle_state = 'deleted'` | `Delete` (remove document) |
| Archive | `UPDATE lifecycle_state = 'archived'` | `Index` (update lifecycle_state) |

**Trade-offs of synchronous dual-write**:

- Pro: Simple implementation, low operational overhead, no additional infrastructure.
- Pro: Tight latency -- ES is updated within the same request.
- Con: Adds latency to every write (ES round-trip time).
- Con: No retry mechanism -- a failed ES write is logged and lost until reconciliation.
- Con: Tight coupling -- the application service must know about ES.

### Future: CDC via Debezium

The target architecture replaces synchronous dual-write with Change Data Capture:

```
PostgreSQL WAL
    |
    v
Debezium Connector (reads WAL)
    |
    v
Kafka Topic (crm.public.contacts, crm.public.opportunities, ...)
    |
    +---> ES Consumer (transforms + indexes)
    |
    +---> Analytics Consumer (loads to ClickHouse/BigQuery)
    |
    +---> Audit Consumer (writes to audit log)
```

Benefits of the CDC approach:

- **Decoupled**: The application service only writes to PG. It has no knowledge of downstream consumers.
- **Reliable**: Kafka provides at-least-once delivery with offset tracking. Failed ES writes are retried automatically.
- **Scalable**: Multiple consumers can process the same change stream independently.
- **Complete**: Every change to PG (including direct SQL modifications, migrations, bulk updates) is captured.

The ES consumer would:

1. Read change events from Kafka.
2. Resolve denormalized fields (e.g., look up `company_name` from the companies topic or a local cache).
3. Transform the event into the ES document shape.
4. Bulk-index to ES using the `_bulk` API for throughput.

### Backfill: Bulk Re-Index from PostgreSQL

When bootstrapping a new ES index or recovering from significant drift, a full re-index is performed:

```
PostgreSQL
    |
    |--- SELECT * FROM contacts
    |    WHERE tenant_id = ? AND lifecycle_state != 'deleted'
    |    ORDER BY id
    |    LIMIT 1000 OFFSET ?
    |
    v
Bulk Transform (compute full_name, resolve company_name, etc.)
    |
    v
ES _bulk API (index 1000 docs per batch)
    |
    v
Repeat until all records processed
```

For large re-indexes (millions of records):

- Use keyset pagination (`WHERE id > last_seen_id ORDER BY id LIMIT 1000`) instead of OFFSET for consistent performance.
- Disable ES `refresh_interval` during bulk load (`"refresh_interval": "-1"`), then restore it after.
- Use multiple worker goroutines to parallelize bulk API calls.
- Create the new index with an alias, then swap the alias atomically after the re-index completes (zero-downtime re-index).

---

## 6. PostgreSQL Indexing Strategy

All indexes in the system are defined in `migrations/000007_create_indexes.up.sql`. The guiding principles:

1. **`tenant_id` is always the leading column** in composite indexes, aligning with the RLS policy `WHERE tenant_id = current_setting('app.current_tenant_id')::uuid`. The query planner can use the index for both the RLS filter and the application-level filter in a single scan.

2. **Index only what queries need** -- every index corresponds to an actual query pattern in the application.

### Composite B-Tree Indexes

```sql
-- Contacts
CREATE INDEX idx_contacts_location_lifecycle ON contacts(tenant_id, lifecycle_state);
CREATE INDEX idx_contacts_location_email ON contacts(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX idx_contacts_location_company ON contacts(tenant_id, company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_contacts_location_created ON contacts(tenant_id, created_at DESC);

-- Opportunities
CREATE INDEX idx_opps_location_lifecycle ON opportunities(tenant_id, lifecycle_state);
CREATE INDEX idx_opps_location_pipeline_stage ON opportunities(tenant_id, pipeline_id, stage_id);
CREATE INDEX idx_opps_location_assigned ON opportunities(tenant_id, assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_opps_location_close_date ON opportunities(tenant_id, expected_close_date)
    WHERE expected_close_date IS NOT NULL;

-- Companies
CREATE INDEX idx_companies_location_lifecycle ON companies(tenant_id, lifecycle_state);
CREATE INDEX idx_companies_location_domain ON companies(tenant_id, domain) WHERE domain IS NOT NULL;

-- Custom Object Records
CREATE INDEX idx_cor_location_schema ON custom_object_records(tenant_id, schema_id);
CREATE INDEX idx_cor_location_lifecycle ON custom_object_records(tenant_id, lifecycle_state);
```

**Index design rationale**:

- `(tenant_id, lifecycle_state)`: Every list query filters by tenant and active state. This is the most frequently used index.
- `(tenant_id, email)`: Contact lookup by email within a tenant. Partial index excludes NULLs (contacts without email).
- `(tenant_id, pipeline_id, stage_id)`: Opportunity board view -- "show all deals in stage X of pipeline Y for this tenant." The three-column composite allows the planner to satisfy the query with an index-only scan.
- `(tenant_id, created_at DESC)`: Supports "most recent contacts" queries with efficient descending scan.
- `(tenant_id, assigned_to)`: Opportunity assignment queries -- "show all deals assigned to user X." Partial index excludes unassigned opportunities.

### GIN Indexes for JSONB and Arrays

```sql
-- JSONB custom properties
CREATE INDEX idx_contacts_custom_props ON contacts USING GIN(custom_properties jsonb_path_ops);
CREATE INDEX idx_companies_custom_props ON companies USING GIN(custom_properties jsonb_path_ops);
CREATE INDEX idx_opps_custom_props ON opportunities USING GIN(custom_properties jsonb_path_ops);
CREATE INDEX idx_cor_properties ON custom_object_records USING GIN(properties jsonb_path_ops);

-- Array columns
CREATE INDEX idx_contacts_tags ON contacts USING GIN(tags);
```

**GIN index rationale**:

- `jsonb_path_ops` is used instead of the default GIN operator class because it produces a smaller index and supports the `@>` containment operator, which is the primary query pattern for JSONB (e.g., `custom_properties @> '{"lead_score": 85}'`).
- The `tags` GIN index supports `@>` (array contains) and `&&` (array overlap) queries, enabling both "contacts with tag X" and "contacts with any of tags [X, Y, Z]" queries.

### Partial Indexes

```sql
WHERE email IS NOT NULL      -- idx_contacts_location_email
WHERE company_id IS NOT NULL -- idx_contacts_location_company
WHERE domain IS NOT NULL     -- idx_companies_location_domain
WHERE assigned_to IS NOT NULL -- idx_opps_location_assigned
WHERE expected_close_date IS NOT NULL -- idx_opps_location_close_date
```

Partial indexes reduce index size and write amplification by excluding rows that would never match the query pattern. For example, there is no value in indexing contacts without an email address in the email lookup index.

### RLS Alignment

Every RLS policy in the system uses the same predicate:

```sql
USING (tenant_id = current_setting('app.current_tenant_id')::uuid)
```

Because `tenant_id` is the leading column in every composite index, the query planner efficiently satisfies both the RLS filter and the application query in a single index scan. Without this alignment, the RLS filter would require a separate sequential scan or filter step.

---

## 7. Partitioning Strategy

### Current State: No Partitioning

The current schema does not use table partitioning. For the expected data volumes in the near term (tens of millions of rows across all tenants), properly indexed tables perform well without partitioning. Adding partitioning prematurely introduces:

- Operational complexity (partition management, maintenance scripts)
- Query planner overhead (partition pruning)
- Migration complexity (converting existing tables to partitioned tables)

**Threshold for partitioning**: When any single table exceeds 100 million rows, or when index maintenance (VACUUM, REINDEX) windows become unacceptable.

### Future: Hash Partitioning by `tenant_id`

For large tables (`contacts`, `custom_object_records`), hash partitioning by `tenant_id` is the recommended strategy:

```sql
-- Example: 16 hash partitions
CREATE TABLE contacts (
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    ...
) PARTITION BY HASH (tenant_id);

CREATE TABLE contacts_p0 PARTITION OF contacts FOR VALUES WITH (MODULUS 16, REMAINDER 0);
CREATE TABLE contacts_p1 PARTITION OF contacts FOR VALUES WITH (MODULUS 16, REMAINDER 1);
-- ... through contacts_p15
```

Benefits:

- **Even distribution**: Hash partitioning distributes tenants uniformly across partitions, avoiding hot spots.
- **Query pruning**: Queries with `tenant_id = ?` (which is every query due to RLS) hit exactly one partition.
- **Parallel maintenance**: VACUUM and REINDEX can run on individual partitions without locking the entire table.
- **Partition-level operations**: Bulk deletion of a tenant's data can be done by detaching and dropping the partition (if using list partitioning per tenant).

### Future: Time-Based Partitioning for Analytics and Audit

Tables that grow unboundedly with time (audit logs, activity streams, analytics events) should be partitioned by time:

```sql
CREATE TABLE audit_log (
    id          UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type  VARCHAR(100) NOT NULL,
    payload     JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

CREATE TABLE audit_log_2025_q4 PARTITION OF audit_log
    FOR VALUES FROM ('2025-10-01') TO ('2026-01-01');
CREATE TABLE audit_log_2026_q1 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');
```

Benefits:

- **Retention policies**: Drop old partitions instead of running expensive DELETE queries.
- **Query performance**: Time-bounded queries (common in reporting) only scan relevant partitions.
- **Tiered storage**: Older partitions can be moved to cheaper storage (tablespaces on slower disks).

### Whale Tenant Handling

For tenants with disproportionately large datasets (>10 million records in a single table):

1. **Detection**: Monitor per-tenant row counts via `pg_stat_user_tables` and application-level counters.
2. **Dedicated partitions**: Use list partitioning to give whale tenants their own partition:
   ```sql
   CREATE TABLE contacts_whale_tenant_abc PARTITION OF contacts
       FOR VALUES IN ('abc-tenant-uuid');
   ```
3. **Separate connection pools**: Route whale tenant queries to read replicas to avoid impacting other tenants.
4. **ES shard splitting**: If a single shard grows too large due to a whale tenant (>50GB), use the ES split index API or re-index with more shards.

---

## 8. Multi-Tenant Search

### Shared Index with Mandatory Tenant Filter

All tenants share the same set of ES indexes (`contacts`, `companies`, `opportunities`, `custom_objects`). Every search query includes a mandatory `tenant_id` term filter:

```go
// From search_repo.go -- buildESQuery
must := []map[string]interface{}{
    {"term": map[string]interface{}{"tenant_id": tenantID}},
}
```

This filter is injected at the infrastructure layer, not by the caller. The application service passes the `tenant_id` via the Go context, and the search repository extracts it and includes it in every query. There is no code path that can execute a search without this filter.

### Routing for Shard Efficiency

ES routing is configured as required (`"_routing": { "required": true }`). The routing value is always `tenant_id`:

```go
r.client.Search.WithRouting(tenantID)
```

Without routing, a search query would fan out to all shards (scatter-gather), then merge results. With routing:

- The query targets exactly 1 shard (deterministic hash of `tenant_id`).
- Latency is reduced by a factor proportional to the number of shards.
- Cluster load is reduced because N-1 shards are not involved.

### Shard Sizing and Monitoring

Target shard size: 10-30 GB per shard. With 3 shards per index:

| Scenario | Total Data | Per Shard | Action |
|---|---|---|---|
| Small deployment | <30 GB | <10 GB | No action needed |
| Medium deployment | 30-90 GB | 10-30 GB | Optimal range |
| Large deployment | 90-300 GB | 30-100 GB | Increase to 6-9 shards |
| Very large deployment | >300 GB | >100 GB | Consider index-per-month or split |

Monitoring checks:

- **Shard size**: `GET _cat/shards?v` -- alert if any shard exceeds 50 GB.
- **Per-tenant document count**: Periodic aggregation to detect tenants whose data dominates a shard.
- **Search latency P99**: Alert if search latency exceeds 500ms, which may indicate hot shards.

### Why Index-Per-Tenant Was Rejected

An alternative approach would create a separate ES index for each tenant (e.g., `contacts_tenant_abc`, `contacts_tenant_def`). This was evaluated and rejected for the following reasons:

| Concern | Index-Per-Tenant | Shared Index with Routing |
|---|---|---|
| **Shard count** | N tenants x M indexes x S shards = thousands of shards | M indexes x S shards = small fixed number |
| **Cluster state size** | Grows linearly with tenants, causing master node pressure | Constant |
| **Operational overhead** | Mapping updates must be applied to every tenant index | Applied once per index |
| **Tenant onboarding** | Requires creating indexes at signup time | Zero-config |
| **Cross-tenant queries** | Requires multi-index queries for platform analytics | Single query with no tenant_id filter |
| **Resource utilization** | Many small shards waste memory (each shard has fixed overhead) | Larger, more efficient shards |

For a CRM platform with potentially thousands of tenants, the shared index approach with routing-based isolation is the correct choice. The shard explosion problem of index-per-tenant is well-documented in Elasticsearch best practices and becomes critical at scale.

### Security Boundary

The multi-tenant search security model has defense in depth:

1. **Application layer**: The `tenant_id` is extracted from the authenticated request context by middleware (`internal/infrastructure/middleware/tenant.go`) and cannot be spoofed by the client.
2. **Query layer**: Every ES query includes a `term` filter on `tenant_id`, injected by the search repository.
3. **Routing layer**: ES routing ensures the query physically only touches the shard containing the tenant's data.
4. **No direct ES access**: Elasticsearch is not exposed to clients. All access goes through the application API.

---

## Summary

| Engine | Role | Consistency | Latency | Tenant Isolation |
|---|---|---|---|---|
| PostgreSQL | Source of truth | Strong (ACID) | Read-after-write | RLS policies on `tenant_id` |
| Elasticsearch | Search and filter | Eventual (2-5s) | Sub-second search | Routing + term filter on `tenant_id` |
| ClickHouse/BigQuery | Analytics | Eventual (1-5min) | Seconds for aggregations | Tenant-scoped queries / views |

The architecture is designed to evolve: the synchronous dual-write can be replaced with CDC when the operational investment is justified, partitioning can be introduced when data volumes demand it, and the analytics tier can be stood up independently without modifying the write path. Each storage engine serves a specific access pattern, and the clear ownership model (writes always go to PG first) prevents split-brain scenarios and simplifies debugging.
