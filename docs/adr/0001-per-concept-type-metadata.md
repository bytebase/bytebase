# Per-concept type metadata messages, not a unified UDT message

`SchemaMetadata` (proto/store/store/database.proto) models named user-defined types as narrow per-concept messages — `EnumTypeMetadata`, `CompositeTypeMetadata`, and in the future `ObjectTypeMetadata` (Oracle), `TableTypeMetadata` (SQL Server), `DomainMetadata` (PostgreSQL) — each named by the owning engine family's official term. We rejected a unified `UserDefinedTypeMetadata`.

Metadata is persisted as protojson in JSONB (snapshots, changelogs, releases) and unmarshaled with `DiscardUnknown`, so field names and message shapes are frozen forever once shipped: a later rename silently orphans historical data with no error. The choice therefore minimizes the probability of ever wanting to restructure.

## Considered options

- **Unified superset struct** (`kind` discriminator + union of all fields): most fields empty per element, meaning depends on `kind`, no invariant, heterogeneous shapes under one JSONB key.
- **Unified `oneof` payloads**: the payloads are per-concept messages anyway; the wrapper shares only `name`/`comment`/`skip_dump` — too thin to support any shared differ, dump, or sync code.
- **Unified `name` + definition text** (the `FunctionMetadata` pattern): only valid when drop-and-recreate is universally safe. Composite types disprove it — `DROP TYPE` fails when columns depend on the type, so migration must go through attribute-level `ALTER TYPE`, which requires structured attributes.

The concepts share almost no behavior: migration DDL, dump syntax, and sync queries are disjoint per concept (composite: `ALTER TYPE ... ATTRIBUTE`; enum: `ADD VALUE` only; domain: `ALTER DOMAIN`; range/alias/table type: recreate-only; Oracle object: `ALTER TYPE ... CASCADE` + body recompile). The engines' own catalogs reach the same conclusion: PostgreSQL keeps `pg_type` as a name registry with per-flavor side catalogs (`pg_attribute`, `pg_enum`, `pg_range`, `pg_constraint`); Oracle splits `ALL_TYPES` / `ALL_TYPE_ATTRS` / `ALL_TYPE_METHODS` / `ALL_COLL_TYPES`.

## Consequences

- `SchemaMetadata` accretes one field per concept over time, empty for engines lacking the concept — the established pattern (`events`, `packages`, `tasks`, `enum_types`).
- Engines share a message only when the concept genuinely matches (CockroachDB/RisingWave composite types reuse `CompositeTypeMetadata`; OceanBase Oracle mode would reuse `ObjectTypeMetadata`).
- A cross-engine "list all named types" view, if ever needed, is an API-level aggregation over per-concept storage — not storage-level unification.
