# API & Data Contract Design (Including Custom Objects)

## Table of Contents

1. [Design Principles](#1-design-principles)
2. [Complete API Reference](#2-complete-api-reference)
3. [Query & Filter Model](#3-query--filter-model)
4. [Error Response Shape](#4-error-response-shape)
5. [Versioning & Evolution](#5-versioning--evolution)
6. [Pagination](#6-pagination)

---

## 1. Design Principles

### 1.1 Generic Object Pattern

Every CRM entity -- whether built-in or tenant-defined -- is accessed through a single, uniform URL structure:

```
/crm/objects/{object_type}
```

The `object_type` path parameter resolves to one of two categories:

| Category      | Examples                                          | Resolution                           |
|---------------|---------------------------------------------------|--------------------------------------|
| Built-in      | `contacts`, `companies`, `opportunities`, `pipelines` | Routed to typed storage tables       |
| Custom Object | `vehicles`, `properties`, `subscriptions`         | Slug looked up in `custom_object_schemas`; routed to `custom_object_records` |

This means a client that can talk to `/crm/objects/contacts` can talk to `/crm/objects/vehicles` with zero code changes. The shape of every request and response is identical regardless of object type.

### 1.2 Consistent Response Envelope

Every successful response wraps its payload in the same shape. Single-record operations return a flat object. Collection operations return a `ListResponse` envelope:

**Single record:**
```json
{
  "id": "...",
  "object_type": "...",
  "properties": { },
  "lifecycle_state": "active",
  "created_at": "...",
  "updated_at": "..."
}
```

**Collection:**
```json
{
  "results": [ ],
  "total": 0,
  "page": 1,
  "page_size": 20,
  "has_more": false
}
```

Clients can parse any response without knowing the object type in advance. SDKs and code generators benefit from a single `RecordResponse` type.

### 1.3 Properties-Based Model

All fields -- system-defined and custom -- live in a flat `properties` object. There is no structural distinction between a built-in field like `email` and a tenant-defined field like `plan_type`. The server maps known property keys to typed columns (for indexing and validation) and stores unrecognized keys in a `jsonb` column (`custom_properties`).

```json
{
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "email": "jane@acme.com",
    "plan_type": "enterprise"
  }
}
```

For custom objects, every property key is validated against the schema's `fields` definition. For built-in objects, the mapper extracts known keys into typed columns and routes the remainder to `custom_properties`.

### 1.4 Tenant & User Context

Multi-tenancy is enforced at the middleware layer. Two headers carry identity:

| Header          | Required | Purpose                                              |
|-----------------|----------|------------------------------------------------------|
| `X-Tenant-Id` | Yes      | Tenant identifier. Every query is scoped to this value. Requests without it receive `400 MISSING_TENANT`. |
| `X-User-Id`     | No       | Authenticated user. Written to `created_by` / `updated_by` audit columns when present. |

These headers are extracted by the `TenantExtractor` middleware and injected into the request context before any handler executes. All repository queries include a `WHERE tenant_id = ?` clause, and Elasticsearch queries include a `term` filter on `tenant_id` with routing by the same value.

---

## 2. Complete API Reference

All paths are relative to the base URL. In v1: `https://api.example.com/crm/...`

Common headers for every request:

```
X-Tenant-Id: d290f1ee-6c54-4b01-90e6-d701748f0851
X-User-Id: 7c9e6679-7425-40de-944b-e07fc1f90ae7       (optional)
Content-Type: application/json
```

---

### 2.1 Generic Object CRUD

#### POST /crm/objects/{object_type}

Create a new record of the given object type.

**Request:**
```json
{
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "email": "jane@acme.com",
    "phone": "+14155551234",
    "source": "website",
    "tags": ["enterprise", "inbound"],
    "plan_type": "enterprise"
  }
}
```

**Response: `201 Created`**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "object_type": "contacts",
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "email": "jane@acme.com",
    "phone": "+14155551234",
    "source": "website",
    "tags": ["enterprise", "inbound"],
    "plan_type": "enterprise"
  },
  "lifecycle_state": "active",
  "created_at": "2026-03-06T10:30:00Z",
  "updated_at": "2026-03-06T10:30:00Z"
}
```

---

#### GET /crm/objects/{object_type}

List records with offset-based pagination.

**Query Parameters:**

| Parameter   | Type | Default | Max | Description                |
|-------------|------|---------|-----|----------------------------|
| `page`      | int  | 1       | --  | Page number (1-indexed)    |
| `page_size` | int  | 20      | 100 | Records per page           |

**Request:**
```
GET /crm/objects/contacts?page=2&page_size=10
```

**Response: `200 OK`**
```json
{
  "results": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "object_type": "contacts",
      "properties": {
        "first_name": "Jane",
        "last_name": "Doe",
        "email": "jane@acme.com"
      },
      "lifecycle_state": "active",
      "created_at": "2026-03-06T10:30:00Z",
      "updated_at": "2026-03-06T10:30:00Z"
    },
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "object_type": "contacts",
      "properties": {
        "first_name": "John",
        "last_name": "Smith",
        "email": "john@globex.com"
      },
      "lifecycle_state": "active",
      "created_at": "2026-03-05T14:22:00Z",
      "updated_at": "2026-03-05T14:22:00Z"
    }
  ],
  "total": 87,
  "page": 2,
  "page_size": 10,
  "has_more": true
}
```

---

#### GET /crm/objects/{object_type}/{id}

Retrieve a single record by ID.

**Request:**
```
GET /crm/objects/contacts/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Response: `200 OK`**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "object_type": "contacts",
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "email": "jane@acme.com",
    "phone": "+14155551234",
    "source": "website",
    "tags": ["enterprise", "inbound"],
    "plan_type": "enterprise"
  },
  "lifecycle_state": "active",
  "created_at": "2026-03-06T10:30:00Z",
  "updated_at": "2026-03-06T10:30:00Z"
}
```

---

#### PATCH /crm/objects/{object_type}/{id}

Partial update. Only the properties supplied in the request body are modified. Omitted properties are left unchanged.

**Request:**
```json
{
  "properties": {
    "phone": "+14155559999",
    "plan_type": "professional"
  }
}
```

**Response: `200 OK`**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "object_type": "contacts",
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "email": "jane@acme.com",
    "phone": "+14155559999",
    "source": "website",
    "tags": ["enterprise", "inbound"],
    "plan_type": "professional"
  },
  "lifecycle_state": "active",
  "created_at": "2026-03-06T10:30:00Z",
  "updated_at": "2026-03-06T11:15:00Z"
}
```

---

#### DELETE /crm/objects/{object_type}/{id}

Soft-delete a record. Sets `lifecycle_state` to `deleted` and populates `deleted_at`. The record is excluded from default list and search queries but remains in the database for audit purposes.

**Request:**
```
DELETE /crm/objects/contacts/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Response: `204 No Content`**

No response body.

---

#### PATCH /crm/objects/{object_type}/{id}/archive

Archive a record. Sets `lifecycle_state` to `archived`. Archived records do not appear in default listings but can be queried explicitly by filtering on `lifecycle_state`.

**Request:**
```
PATCH /crm/objects/contacts/a1b2c3d4-e5f6-7890-abcd-ef1234567890/archive
```

**Response: `200 OK`**
```json
{
  "status": "archived"
}
```

---

#### PATCH /crm/objects/{object_type}/{id}/restore

Restore a previously archived or soft-deleted record to `active` state.

**Request:**
```
PATCH /crm/objects/contacts/a1b2c3d4-e5f6-7890-abcd-ef1234567890/restore
```

**Response: `200 OK`**
```json
{
  "status": "restored"
}
```

---

### 2.2 Search

#### POST /crm/objects/{object_type}/search

Full-text and structured search. The search endpoint delegates to Elasticsearch for execution and returns results in the standard list envelope.

**Request:**
```json
{
  "filters": [
    {
      "operator": "AND",
      "conditions": [
        { "field": "lifecycle_state", "operator": "eq", "value": "active" },
        { "field": "source", "operator": "in", "value": ["website", "referral"] }
      ]
    }
  ],
  "sort": [
    { "field": "created_at", "direction": "desc" }
  ],
  "query": "Jane",
  "page": 1,
  "page_size": 20
}
```

**Response: `200 OK`**
```json
{
  "results": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "object_type": "contacts",
      "properties": {
        "first_name": "Jane",
        "last_name": "Doe",
        "email": "jane@acme.com",
        "source": "website"
      },
      "lifecycle_state": "active",
      "created_at": "2026-03-06T10:30:00Z",
      "updated_at": "2026-03-06T10:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20,
  "has_more": false
}
```

See [Section 3: Query & Filter Model](#3-query--filter-model) for the full filter specification.

---

### 2.3 Associations

Associations create typed, bidirectional links between records. Each association references an `AssociationDefinition` that declares the allowed object types and cardinality.

#### POST /crm/objects/{object_type}/{id}/associations

Create an association from the source record to a target record.

**Request:**
```json
{
  "definition_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "target_record_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "target_object_type": "companies"
}
```

**Response: `201 Created`**
```json
{
  "id": "d4e5f6a7-b8c9-0123-def0-1234567890ab",
  "definition_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "source_record_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "target_record_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "source_object_type": "contacts",
  "target_object_type": "companies",
  "created_at": "2026-03-06T11:00:00Z"
}
```

---

#### GET /crm/objects/{object_type}/{id}/associations

List all associations for a given record.

**Query Parameters:**

| Parameter   | Type | Default | Description            |
|-------------|------|---------|------------------------|
| `page`      | int  | 1       | Page number            |
| `page_size` | int  | 20      | Associations per page  |

**Request:**
```
GET /crm/objects/contacts/a1b2c3d4-e5f6-7890-abcd-ef1234567890/associations?page=1&page_size=10
```

**Response: `200 OK`**
```json
{
  "results": [
    {
      "id": "d4e5f6a7-b8c9-0123-def0-1234567890ab",
      "definition_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "source_record_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "target_record_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "source_object_type": "contacts",
      "target_object_type": "companies",
      "created_at": "2026-03-06T11:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10
}
```

---

#### DELETE /crm/objects/{object_type}/{id}/associations/{assoc_id}

Remove an association.

**Request:**
```
DELETE /crm/objects/contacts/a1b2c3d4-e5f6-7890-abcd-ef1234567890/associations/d4e5f6a7-b8c9-0123-def0-1234567890ab
```

**Response: `204 No Content`**

No response body.

---

### 2.4 Custom Object Schemas

Schemas define the structure of a custom object type: its name, slug (used as the `object_type` in the generic endpoint), and the field definitions that govern validation and indexing.

#### POST /crm/schemas

Create a new custom object schema.

**Request:**
```json
{
  "singular_name": "Vehicle",
  "plural_name": "Vehicles",
  "slug": "vehicles",
  "primary_field": "vin",
  "fields": [
    {
      "key": "vin",
      "label": "VIN",
      "field_type": "text",
      "required": true,
      "unique": true
    },
    {
      "key": "make",
      "label": "Make",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "model",
      "label": "Model",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "year",
      "label": "Year",
      "field_type": "number",
      "required": true,
      "unique": false
    },
    {
      "key": "color",
      "label": "Color",
      "field_type": "dropdown",
      "required": false,
      "unique": false,
      "options": ["red", "blue", "black", "white", "silver", "other"]
    }
  ]
}
```

**Response: `201 Created`**
```json
{
  "id": "e5f6a7b8-c9d0-1234-ef01-234567890abc",
  "slug": "vehicles",
  "singular_name": "Vehicle",
  "plural_name": "Vehicles",
  "primary_field": "vin",
  "fields": [
    {
      "key": "vin",
      "label": "VIN",
      "field_type": "text",
      "required": true,
      "unique": true
    },
    {
      "key": "make",
      "label": "Make",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "model",
      "label": "Model",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "year",
      "label": "Year",
      "field_type": "number",
      "required": true,
      "unique": false
    },
    {
      "key": "color",
      "label": "Color",
      "field_type": "dropdown",
      "required": false,
      "unique": false,
      "options": ["red", "blue", "black", "white", "silver", "other"]
    }
  ],
  "lifecycle_state": "active",
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z"
}
```

After creation, records of this type can be managed via `POST /crm/objects/vehicles`, `GET /crm/objects/vehicles`, and so on.

**Supported `field_type` values:** `text`, `textarea`, `number`, `date`, `phone`, `email`, `dropdown`, `boolean`.

---

#### GET /crm/schemas

List all custom object schemas for the tenant.

**Query Parameters:**

| Parameter   | Type | Default | Description          |
|-------------|------|---------|----------------------|
| `page`      | int  | 1       | Page number          |
| `page_size` | int  | 20      | Schemas per page     |

**Request:**
```
GET /crm/schemas?page=1&page_size=10
```

**Response: `200 OK`**
```json
{
  "results": [
    {
      "id": "e5f6a7b8-c9d0-1234-ef01-234567890abc",
      "slug": "vehicles",
      "singular_name": "Vehicle",
      "plural_name": "Vehicles",
      "primary_field": "vin",
      "fields": [ ],
      "lifecycle_state": "active",
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10
}
```

---

#### GET /crm/schemas/{id}

Retrieve a single schema by ID.

**Request:**
```
GET /crm/schemas/e5f6a7b8-c9d0-1234-ef01-234567890abc
```

**Response: `200 OK`**
```json
{
  "id": "e5f6a7b8-c9d0-1234-ef01-234567890abc",
  "slug": "vehicles",
  "singular_name": "Vehicle",
  "plural_name": "Vehicles",
  "primary_field": "vin",
  "fields": [
    {
      "key": "vin",
      "label": "VIN",
      "field_type": "text",
      "required": true,
      "unique": true
    },
    {
      "key": "make",
      "label": "Make",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "model",
      "label": "Model",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "year",
      "label": "Year",
      "field_type": "number",
      "required": true,
      "unique": false
    },
    {
      "key": "color",
      "label": "Color",
      "field_type": "dropdown",
      "required": false,
      "unique": false,
      "options": ["red", "blue", "black", "white", "silver", "other"]
    }
  ],
  "lifecycle_state": "active",
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z"
}
```

---

#### PATCH /crm/schemas/{id}

Update an existing schema. Typically used to add new fields or modify labels. Removing a field that has existing data should be handled carefully (the field can be marked deprecated rather than deleted).

**Request:**
```json
{
  "singular_name": "Vehicle",
  "plural_name": "Vehicles",
  "slug": "vehicles",
  "primary_field": "vin",
  "fields": [
    {
      "key": "vin",
      "label": "VIN",
      "field_type": "text",
      "required": true,
      "unique": true
    },
    {
      "key": "make",
      "label": "Make",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "model",
      "label": "Model",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "year",
      "label": "Year",
      "field_type": "number",
      "required": true,
      "unique": false
    },
    {
      "key": "color",
      "label": "Color",
      "field_type": "dropdown",
      "required": false,
      "unique": false,
      "options": ["red", "blue", "black", "white", "silver", "other"]
    },
    {
      "key": "mileage",
      "label": "Mileage",
      "field_type": "number",
      "required": false,
      "unique": false
    }
  ]
}
```

**Response: `200 OK`**
```json
{
  "id": "e5f6a7b8-c9d0-1234-ef01-234567890abc",
  "slug": "vehicles",
  "singular_name": "Vehicle",
  "plural_name": "Vehicles",
  "primary_field": "vin",
  "fields": [
    {
      "key": "vin",
      "label": "VIN",
      "field_type": "text",
      "required": true,
      "unique": true
    },
    {
      "key": "make",
      "label": "Make",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "model",
      "label": "Model",
      "field_type": "text",
      "required": true,
      "unique": false
    },
    {
      "key": "year",
      "label": "Year",
      "field_type": "number",
      "required": true,
      "unique": false
    },
    {
      "key": "color",
      "label": "Color",
      "field_type": "dropdown",
      "required": false,
      "unique": false,
      "options": ["red", "blue", "black", "white", "silver", "other"]
    },
    {
      "key": "mileage",
      "label": "Mileage",
      "field_type": "number",
      "required": false,
      "unique": false
    }
  ],
  "lifecycle_state": "active",
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:45:00Z"
}
```

---

#### DELETE /crm/schemas/{id}

Delete a custom object schema. This soft-deletes the schema and prevents new records from being created under this object type. Existing records remain in the database but become inaccessible through the generic object endpoints.

**Request:**
```
DELETE /crm/schemas/e5f6a7b8-c9d0-1234-ef01-234567890abc
```

**Response: `204 No Content`**

No response body.

---

### 2.5 Association Definitions

Association definitions declare the rules for linking two object types. They specify cardinality and provide human-readable labels for each side of the relationship.

**Supported cardinality values:** `one_to_one`, `one_to_many`, `many_to_many`.

#### POST /crm/association-definitions

Create a new association definition.

**Request:**
```json
{
  "source_object_type": "contacts",
  "target_object_type": "companies",
  "source_label": "Works At",
  "target_label": "Employs",
  "cardinality": "many_to_many"
}
```

**Response: `201 Created`**
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "source_object_type": "contacts",
  "target_object_type": "companies",
  "source_label": "Works At",
  "target_label": "Employs",
  "cardinality": "many_to_many",
  "created_at": "2026-03-06T11:00:00Z"
}
```

Association definitions work across built-in and custom object types. For example, you can define a `contacts <-> vehicles` relationship once a `vehicles` schema exists:

```json
{
  "source_object_type": "contacts",
  "target_object_type": "vehicles",
  "source_label": "Owns",
  "target_label": "Owned By",
  "cardinality": "one_to_many"
}
```

---

#### GET /crm/association-definitions

List all association definitions for the tenant.

**Query Parameters:**

| Parameter   | Type | Default | Description              |
|-------------|------|---------|--------------------------|
| `page`      | int  | 1       | Page number              |
| `page_size` | int  | 20      | Definitions per page     |

**Request:**
```
GET /crm/association-definitions?page=1&page_size=10
```

**Response: `200 OK`**
```json
{
  "results": [
    {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "source_object_type": "contacts",
      "target_object_type": "companies",
      "source_label": "Works At",
      "target_label": "Employs",
      "cardinality": "many_to_many",
      "created_at": "2026-03-06T11:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10
}
```

---

#### DELETE /crm/association-definitions/{id}

Delete an association definition. Existing associations referencing this definition are not automatically removed but become orphaned. Clients should clean up related associations before or after deletion.

**Request:**
```
DELETE /crm/association-definitions/f47ac10b-58cc-4372-a567-0e02b2c3d479
```

**Response: `204 No Content`**

No response body.

---

### 2.6 Pipelines

Pipelines represent multi-stage workflows (e.g., sales funnels, deal stages). Opportunities reference a pipeline and a specific stage within it.

#### POST /crm/pipelines

Create a pipeline with an initial set of stages.

**Request:**
```json
{
  "name": "Enterprise Sales",
  "stages": [
    { "name": "Prospecting", "position": 0 },
    { "name": "Qualification", "position": 1 },
    { "name": "Proposal", "position": 2 },
    { "name": "Negotiation", "position": 3 },
    { "name": "Closed Won", "position": 4 },
    { "name": "Closed Lost", "position": 5 }
  ]
}
```

**Response: `201 Created`**
```json
{
  "id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "name": "Enterprise Sales",
  "lifecycle_state": "active",
  "stages": [
    {
      "id": "aa111111-1111-1111-1111-111111111111",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Prospecting",
      "position": 0,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "bb222222-2222-2222-2222-222222222222",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Qualification",
      "position": 1,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "cc333333-3333-3333-3333-333333333333",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Proposal",
      "position": 2,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "dd444444-4444-4444-4444-444444444444",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Negotiation",
      "position": 3,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "ee555555-5555-5555-5555-555555555555",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Closed Won",
      "position": 4,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "ff666666-6666-6666-6666-666666666666",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Closed Lost",
      "position": 5,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    }
  ],
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z"
}
```

---

#### GET /crm/pipelines

List all pipelines for the tenant.

**Query Parameters:**

| Parameter   | Type | Default | Description           |
|-------------|------|---------|-----------------------|
| `page`      | int  | 1       | Page number           |
| `page_size` | int  | 20      | Pipelines per page    |

**Request:**
```
GET /crm/pipelines?page=1&page_size=10
```

**Response: `200 OK`**
```json
{
  "results": [
    {
      "id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "object_type": "pipelines",
      "properties": {
        "name": "Enterprise Sales"
      },
      "lifecycle_state": "active",
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "has_more": false
}
```

---

#### GET /crm/pipelines/{id}

Get a pipeline with all of its stages.

**Request:**
```
GET /crm/pipelines/1a2b3c4d-5e6f-7890-abcd-ef1234567890
```

**Response: `200 OK`**
```json
{
  "id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
  "name": "Enterprise Sales",
  "lifecycle_state": "active",
  "stages": [
    {
      "id": "aa111111-1111-1111-1111-111111111111",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Prospecting",
      "position": 0,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    },
    {
      "id": "bb222222-2222-2222-2222-222222222222",
      "pipeline_id": "1a2b3c4d-5e6f-7890-abcd-ef1234567890",
      "name": "Qualification",
      "position": 1,
      "created_at": "2026-03-06T12:00:00Z",
      "updated_at": "2026-03-06T12:00:00Z"
    }
  ],
  "created_at": "2026-03-06T12:00:00Z",
  "updated_at": "2026-03-06T12:00:00Z"
}
```

---

#### POST /crm/pipelines/{id}/stages

Add a stage to an existing pipeline.

**Request:**
```json
{
  "name": "Contract Review",
  "position": 3
}
```

**Response: `201 Created`**
```json
{
  "status": "stage added"
}
```

---

## 3. Query & Filter Model

The search endpoint (`POST /crm/objects/{object_type}/search`) accepts a structured query body that supports boolean filter groups, sorting, full-text search, and pagination.

### 3.1 Request Schema

```json
{
  "filters": [
    {
      "operator": "AND",
      "conditions": [
        { "field": "lifecycle_state", "operator": "eq", "value": "active" },
        { "field": "source", "operator": "in", "value": ["website", "referral"] },
        { "field": "custom_properties.plan_type", "operator": "eq", "value": "enterprise" }
      ]
    }
  ],
  "sort": [
    { "field": "created_at", "direction": "desc" }
  ],
  "query": "Jane",
  "page": 1,
  "page_size": 20
}
```

### 3.2 Filter Structure

| Field           | Type            | Description                                                  |
|-----------------|-----------------|--------------------------------------------------------------|
| `filters`       | `FilterGroup[]` | Array of filter groups. Groups are combined with AND at the top level. |
| `filters[].operator` | `string`  | Boolean operator within the group: `AND` or `OR`.            |
| `filters[].conditions` | `FilterCondition[]` | Array of individual conditions within the group.  |
| `sort`          | `SortClause[]`  | Ordered list of sort directives.                             |
| `query`         | `string`        | Free-text search string. Matched against searchable fields via multi_match. |
| `page`          | `int`           | Page number, 1-indexed. Defaults to 1.                       |
| `page_size`     | `int`           | Results per page. Defaults to 20. Maximum 100.               |

### 3.3 Supported Operators

| Operator   | Description                   | Value Type        | Example                                                           |
|------------|-------------------------------|-------------------|-------------------------------------------------------------------|
| `eq`       | Equals                        | scalar            | `{ "field": "status", "operator": "eq", "value": "active" }`     |
| `neq`      | Not equals                    | scalar            | `{ "field": "status", "operator": "neq", "value": "deleted" }`   |
| `gt`       | Greater than                  | number / date     | `{ "field": "monetary_value", "operator": "gt", "value": 10000 }` |
| `gte`      | Greater than or equal         | number / date     | `{ "field": "created_at", "operator": "gte", "value": "2026-01-01" }` |
| `lt`       | Less than                     | number / date     | `{ "field": "year", "operator": "lt", "value": 2020 }`           |
| `lte`      | Less than or equal            | number / date     | `{ "field": "monetary_value", "operator": "lte", "value": 50000 }` |
| `contains` | Substring / full-text match   | string            | `{ "field": "email", "operator": "contains", "value": "acme" }`  |
| `in`       | Value in set                  | array             | `{ "field": "source", "operator": "in", "value": ["web", "api"] }` |
| `between`  | Value in range (inclusive)    | array of 2        | `{ "field": "created_at", "operator": "between", "value": ["2026-01-01", "2026-03-01"] }` |

### 3.4 Nested Filter Groups

Multiple filter groups allow expressing complex boolean logic. Each group's conditions are combined with the group's `operator` (AND or OR). The groups themselves are combined at the top level with AND:

```json
{
  "filters": [
    {
      "operator": "AND",
      "conditions": [
        { "field": "lifecycle_state", "operator": "eq", "value": "active" }
      ]
    },
    {
      "operator": "OR",
      "conditions": [
        { "field": "source", "operator": "eq", "value": "website" },
        { "field": "source", "operator": "eq", "value": "referral" }
      ]
    }
  ]
}
```

This produces: `lifecycle_state = 'active' AND (source = 'website' OR source = 'referral')`.

### 3.5 Mapping to PostgreSQL WHERE Clauses

When search falls back to PostgreSQL (or for direct database queries), filters translate to parameterized SQL:

| Filter Operator | SQL Output                                          |
|-----------------|-----------------------------------------------------|
| `eq`            | `WHERE field = $1`                                  |
| `neq`           | `WHERE field != $1`                                 |
| `gt`            | `WHERE field > $1`                                  |
| `gte`           | `WHERE field >= $1`                                 |
| `lt`            | `WHERE field < $1`                                  |
| `lte`           | `WHERE field <= $1`                                 |
| `contains`      | `WHERE field ILIKE '%' \|\| $1 \|\| '%'`           |
| `in`            | `WHERE field = ANY($1)`                             |
| `between`       | `WHERE field BETWEEN $1 AND $2`                     |

Custom property fields (prefixed with `custom_properties.`) use the JSONB arrow operator:

```sql
WHERE custom_properties->>'plan_type' = $1
```

Group operators map directly:

```sql
-- AND group
WHERE (lifecycle_state = $1 AND source = ANY($2))

-- OR group
WHERE (source = $1 OR source = $2)
```

The tenant scope is always prepended:

```sql
WHERE tenant_id = $tenant AND lifecycle_state != 'deleted' AND (...)
```

### 3.6 Mapping to Elasticsearch Bool Queries

The `buildESQuery` function in the search repository translates the filter model into the Elasticsearch Query DSL. Every query starts with a `bool.must` clause containing the tenant term filter.

**Full mapping:**

| Filter Operator | ES Query DSL                                                                          |
|-----------------|---------------------------------------------------------------------------------------|
| `eq`            | `{ "term": { "field": "value" } }`                                                   |
| `neq`           | `{ "bool": { "must_not": [{ "term": { "field": "value" } }] } }`                     |
| `gt`            | `{ "range": { "field": { "gt": "value" } } }`                                        |
| `gte`           | `{ "range": { "field": { "gte": "value" } } }`                                       |
| `lt`            | `{ "range": { "field": { "lt": "value" } } }`                                        |
| `lte`           | `{ "range": { "field": { "lte": "value" } } }`                                       |
| `contains`      | `{ "match": { "field": "value" } }`                                                  |
| `in`            | `{ "terms": { "field": ["v1", "v2"] } }`                                             |
| `between`       | `{ "range": { "field": { "gte": "v1", "lte": "v2" } } }`                             |

**Group operator mapping:**

- `AND` group: conditions are appended directly to `bool.must`.
- `OR` group: conditions are wrapped in `{ "bool": { "should": [...], "minimum_should_match": 1 } }` and the wrapper is added to `bool.must`.

**Free-text query mapping:**

The `query` field maps to a `multi_match` query across searchable fields:

```json
{
  "multi_match": {
    "query": "Jane",
    "fields": ["first_name", "last_name", "email", "name", "full_name"],
    "type": "best_fields"
  }
}
```

**Complete Elasticsearch query example:**

For a search requesting active contacts from website/referral sources matching "Jane":

```json
{
  "query": {
    "bool": {
      "must": [
        { "term": { "tenant_id": "d290f1ee-6c54-4b01-90e6-d701748f0851" } },
        {
          "multi_match": {
            "query": "Jane",
            "fields": ["first_name", "last_name", "email", "name", "full_name"],
            "type": "best_fields"
          }
        },
        { "term": { "lifecycle_state": "active" } },
        { "terms": { "source": ["website", "referral"] } }
      ]
    }
  },
  "from": 0,
  "size": 20,
  "sort": [
    { "created_at": { "order": "desc" } }
  ]
}
```

---

## 4. Error Response Shape

All error responses follow a consistent envelope:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "human readable message",
    "details": [
      { "field": "properties.email", "message": "invalid email format" },
      { "field": "properties.first_name", "message": "field is required" }
    ]
  }
}
```

### 4.1 Error Codes

| Code               | HTTP Status | Description                                                     |
|--------------------|-------------|-----------------------------------------------------------------|
| `VALIDATION_ERROR` | 400         | Request body failed validation. `details` contains per-field errors. |
| `MISSING_TENANT`   | 400         | The `X-Tenant-Id` header was not provided.                    |
| `INVALID_ID`       | 400         | A path parameter expected a valid UUID but received an invalid value. |
| `NOT_FOUND`        | 404         | The requested resource does not exist or is not visible to the tenant. |
| `INTERNAL_ERROR`   | 500         | An unexpected server-side error occurred.                       |

### 4.2 Error Examples

**Missing tenant header:**
```
HTTP/1.1 400 Bad Request

{
  "error": {
    "code": "MISSING_TENANT",
    "message": "X-Tenant-Id header is required"
  }
}
```

**Invalid UUID in path:**
```
HTTP/1.1 400 Bad Request

{
  "error": {
    "code": "INVALID_ID",
    "message": "invalid record id"
  }
}
```

**Record not found:**
```
HTTP/1.1 404 Not Found

{
  "error": {
    "code": "NOT_FOUND",
    "message": "resource not found"
  }
}
```

**Validation failure on create:**
```
HTTP/1.1 400 Bad Request

{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "request validation failed",
    "details": [
      { "field": "properties.email", "message": "invalid email format" },
      { "field": "properties.first_name", "message": "field is required" }
    ]
  }
}
```

**Internal error:**
```
HTTP/1.1 500 Internal Server Error

{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "an unexpected error occurred"
  }
}
```

---

## 5. Versioning & Evolution

### 5.1 Strategy: URI-Based Versioning

The API version is embedded in the URI path:

```
/v1/crm/objects/{object_type}
/v2/crm/objects/{object_type}
```

Each major version represents a distinct contract. Within a version, changes are backward-compatible: new fields can appear, but existing fields retain their type and semantics.

### 5.2 Backward-Compatible Changes (No Version Bump)

The following changes are made without incrementing the API version:

- **Adding a new property to a response.** Old clients ignore unknown fields. Example: adding a `score` field to the contact properties object.
- **Adding a new optional field to a request.** Existing clients that omit the field continue to work.
- **Adding a new endpoint.** Does not affect existing integrations.
- **Adding a new enum value.** Clients that use a switch/case on known values should have a default handler.

### 5.3 Field Renaming: Alias Strategy

Renaming a field is a multi-step process that avoids breaking existing clients.

**Example: Contact v1 to v2 Evolution**

In v1, contacts have separate `first_name` and `last_name` properties. In v2, the API adds a computed `full_name` property and deprecates the individual fields.

**Step 1 -- Add the new field (v1, backward-compatible):**

The server begins returning `full_name` alongside `first_name` and `last_name`:

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "object_type": "contacts",
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "full_name": "Jane Doe",
    "email": "jane@acme.com"
  },
  "lifecycle_state": "active",
  "created_at": "2026-03-06T10:30:00Z",
  "updated_at": "2026-03-06T10:30:00Z"
}
```

The response includes a deprecation header to signal clients:

```
Deprecation: true
Sunset: 2027-03-06
Link: </v2/crm/objects/contacts>; rel="successor-version"
```

**Step 2 -- Transition period (N versions, typically 6-12 months):**

During the transition:

- `POST` and `PATCH` requests accept both `first_name`/`last_name` and `full_name` as input.
- Responses include all three fields.
- The server computes `full_name` from the parts on read and parses `full_name` into parts on write (if only `full_name` is provided).
- API documentation marks `first_name` and `last_name` as deprecated.

**Step 3 -- New version (v2):**

The v2 endpoint drops `first_name` and `last_name` from responses. Clients that still send them receive a `VALIDATION_WARNING` in the response (non-blocking) or the fields are silently accepted and parsed.

v1 endpoint:
```json
{
  "properties": {
    "first_name": "Jane",
    "last_name": "Doe",
    "full_name": "Jane Doe"
  }
}
```

v2 endpoint:
```json
{
  "properties": {
    "full_name": "Jane Doe",
    "email": "jane@acme.com"
  }
}
```

### 5.4 Breaking Changes

When a change cannot be made backward-compatible (removing a required field, changing a field type, restructuring the response envelope), a new version is created:

| Phase           | Duration       | Description                                                          |
|-----------------|----------------|----------------------------------------------------------------------|
| Announcement    | T              | Deprecation headers added to v1 responses. Changelog published.      |
| Dual running    | 6-12 months    | Both `/v1/crm/...` and `/v2/crm/...` are active. v1 returns deprecation headers. |
| v1 sunset       | After sunset   | v1 returns `410 Gone` with a body pointing to v2.                    |

### 5.5 Response Headers for Deprecation

During the transition period, deprecated endpoints include:

```
Deprecation: true
Sunset: 2027-03-06T00:00:00Z
Link: </v2/crm/objects/contacts>; rel="successor-version"
```

Clients can programmatically detect these headers and plan migration.

---

## 6. Pagination

### 6.1 Current: Offset-Based Pagination

All list endpoints use offset-based pagination with two query parameters:

| Parameter   | Type | Default | Range   | Description                |
|-------------|------|---------|---------|----------------------------|
| `page`      | int  | 1       | >= 1    | 1-indexed page number      |
| `page_size` | int  | 20      | 1 - 100 | Number of records per page |

**Response fields:**

```json
{
  "results": [ ],
  "total": 245,
  "page": 3,
  "page_size": 20,
  "has_more": true
}
```

| Field       | Type    | Description                                                    |
|-------------|---------|----------------------------------------------------------------|
| `results`   | array   | The records for the current page.                              |
| `total`     | int     | Total number of records matching the query (across all pages). |
| `page`      | int     | The current page number (echo of the request).                 |
| `page_size` | int     | The current page size (echo of the request).                   |
| `has_more`  | bool    | `true` if additional pages exist beyond the current one.       |

The server computes `has_more` as `page * page_size < total`.

Internal SQL translation:

```sql
SELECT * FROM contacts
WHERE tenant_id = $1 AND lifecycle_state != 'deleted'
ORDER BY created_at DESC
LIMIT $page_size OFFSET ($page - 1) * $page_size;
```

Elasticsearch translation:

```json
{
  "from": 40,
  "size": 20
}
```

### 6.2 Limitations of Offset-Based Pagination

Offset pagination works well for small-to-medium datasets but has known issues at scale:

- **Performance degradation.** `OFFSET N` in PostgreSQL still scans and discards N rows. At page 500 with page_size 20, the database scans 10,000 rows to return 20. Elasticsearch has a similar `max_result_window` limit (default 10,000).
- **Unstable windows.** If records are inserted or deleted between page fetches, the client may see duplicate records or miss records entirely.

### 6.3 Future: Cursor-Based Pagination

For large datasets and real-time feeds, a cursor-based approach will be introduced:

**Request:**
```
GET /crm/objects/contacts?page_size=20&after=eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wNlQxMDozMDowMFoiLCJpZCI6ImExYjJjM2Q0LWU1ZjYtNzg5MC1hYmNkLWVmMTIzNDU2Nzg5MCJ9
```

**Response:**
```json
{
  "results": [ ],
  "page_size": 20,
  "has_more": true,
  "cursors": {
    "after": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0wNVQxNDoyMjowMFoiLCJpZCI6ImIyYzNkNGU1LWY2YTctODkwMS1iY2RlLWYxMjM0NTY3ODkwMSJ9"
  }
}
```

The `after` parameter is an opaque, base64-encoded cursor that encodes the sort key(s) and the record ID of the last item in the previous page. The server uses a keyset condition rather than an offset:

```sql
SELECT * FROM contacts
WHERE tenant_id = $1
  AND lifecycle_state != 'deleted'
  AND (created_at, id) < ($cursor_created_at, $cursor_id)
ORDER BY created_at DESC, id DESC
LIMIT $page_size;
```

### 6.4 Why Cursor Pagination is Superior at Scale

| Concern               | Offset                                | Cursor                                   |
|-----------------------|---------------------------------------|------------------------------------------|
| Query performance     | O(offset + limit) -- scans skipped rows | O(limit) -- seeks directly via index     |
| Result stability      | Unstable under concurrent writes      | Stable -- keyset condition is deterministic |
| Deep page access      | Degrades linearly                     | Constant time regardless of position     |
| Elasticsearch window  | Subject to `max_result_window`        | Uses `search_after`, no window limit     |
| Backward compatibility | N/A (current default)               | Additive -- `after` parameter is optional |

The cursor-based approach will be introduced as an additive, non-breaking change: if the `after` parameter is present, cursor mode is used; if only `page` and `page_size` are present, offset mode continues to work. This ensures full backward compatibility during the transition.

---

## Appendix: Endpoint Summary

| Method   | Path                                                         | Description                    |
|----------|--------------------------------------------------------------|--------------------------------|
| `POST`   | `/crm/objects/{object_type}`                                 | Create record                  |
| `GET`    | `/crm/objects/{object_type}`                                 | List records                   |
| `GET`    | `/crm/objects/{object_type}/{id}`                            | Get record by ID               |
| `PATCH`  | `/crm/objects/{object_type}/{id}`                            | Update record                  |
| `DELETE` | `/crm/objects/{object_type}/{id}`                            | Soft-delete record             |
| `PATCH`  | `/crm/objects/{object_type}/{id}/archive`                    | Archive record                 |
| `PATCH`  | `/crm/objects/{object_type}/{id}/restore`                    | Restore record                 |
| `POST`   | `/crm/objects/{object_type}/search`                          | Search/filter records          |
| `POST`   | `/crm/objects/{object_type}/{id}/associations`               | Create association             |
| `GET`    | `/crm/objects/{object_type}/{id}/associations`               | List associations              |
| `DELETE` | `/crm/objects/{object_type}/{id}/associations/{assoc_id}`    | Remove association             |
| `POST`   | `/crm/schemas`                                               | Create custom object schema    |
| `GET`    | `/crm/schemas`                                               | List schemas                   |
| `GET`    | `/crm/schemas/{id}`                                          | Get schema                     |
| `PATCH`  | `/crm/schemas/{id}`                                          | Update schema                  |
| `DELETE` | `/crm/schemas/{id}`                                          | Delete schema                  |
| `POST`   | `/crm/association-definitions`                               | Create association definition  |
| `GET`    | `/crm/association-definitions`                               | List association definitions   |
| `DELETE` | `/crm/association-definitions/{id}`                          | Delete association definition  |
| `POST`   | `/crm/pipelines`                                             | Create pipeline with stages    |
| `GET`    | `/crm/pipelines`                                             | List pipelines                 |
| `GET`    | `/crm/pipelines/{id}`                                        | Get pipeline with stages       |
| `POST`   | `/crm/pipelines/{id}/stages`                                 | Add stage to pipeline          |
