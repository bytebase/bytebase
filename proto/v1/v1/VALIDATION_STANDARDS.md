# Validation Standards for Bytebase Proto Definitions

This document defines validation limits for Bytebase proto fields based on **GitHub** and **Google Cloud Platform** standards.

## Standard Validation Tiers

Bytebase uses a **6-tier system** aligned with industry standards:

| Tier       | Size       | Use For                        | Industry Reference     |
| ---------- | ---------- | ------------------------------ | ---------------------- |
| **TINY**   | **64**     | Labels, tags, versions         | Kubernetes labels      |
| **SMALL**  | **200**    | Titles, display names          | GCP display names      |
| **MEDIUM** | **1,000**  | Short descriptions, config     | GCP conditions         |
| **LARGE**  | **10,000** | Long descriptions, JSON config | Practical limit        |
| **XLARGE** | **65,536** | Comments, review text          | GitHub comments (64KB) |

### Quick Decision Guide

```
What are you validating?
│
├─ Label/tag/version?           → TINY (64)
├─ Title/display name?          → SMALL (200)
├─ Short description/config?    → MEDIUM (1,000)
├─ Long description/JSON?       → LARGE (10,000)
├─ Comment/review?              → XLARGE (65,536)
```

## Field Type Reference

### TINY Tier (64 chars)

```proto
string label = 1 [(buf.validate.field).string.max_len = 64];
```

**Use for:**

- Labels and tags: `production`, `critical`, `bug`
- Version strings: `v1.2.3-beta.1+build.123`
- Status codes: `SUCCESS`, `FAILED`

### SMALL Tier (200 chars)

```proto
string title = 1 [(buf.validate.field).string.max_len = 200];
```

**Use for:**

- Titles: Issue titles, group titles, project names
- Display names: User names, database names, instance names
- Resource names: `projects/my-project`, `databases/production-db`

### MEDIUM Tier (1,000 chars)

```proto
string description = 1 [(buf.validate.field).string.max_len = 1000];
```

**Use for:**

- Short descriptions: Group descriptions, project summaries
- Search queries: User search strings
- Filter expressions: Database filters, query conditions
- Configuration values: Environment variables, settings

### LARGE Tier (10,000 chars)

```proto
string content = 1 [(buf.validate.field).string.max_len = 10000];
```

**Use for:**

- Long descriptions: Detailed issue descriptions, plan details
- JSON payloads: Metadata, configuration JSON
- Documentation: Embedded documentation, help text
- Error messages: Detailed error context

### XLARGE Tier (65,536 chars)

```proto
string comment = 1 [(buf.validate.field).string.max_len = 65536];
```

**Use for:**

- Issue comments: User comments on issues
- Review comments: Code review feedback
- Discussion threads: Long-form discussions
- Rich text content: Markdown or formatted text

**Note**: 65,536 = 64KB, matching GitHub's comment limit.

## Common Validation Patterns

### Resource Validation (Recommended for Bytebase)

Following Google AIP-133, validate the resource message itself:

```proto
message Group {
  option (google.api.resource) = {
    type: "bytebase.com/Group"
    pattern: "groups/{group}"
  };

  string name = 1 [(google.api.field_behavior) = OUTPUT_ONLY];

  // Required title (SMALL tier)
  string title = 2 [
    (google.api.field_behavior) = REQUIRED,
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 200
  ];

  // Optional description (MEDIUM tier)
  string description = 3 [(buf.validate.field).string.max_len = 1000];
}

message CreateGroupRequest {
  string parent = 1 [(google.api.field_behavior) = REQUIRED];
  Group group = 2 [(google.api.field_behavior) = REQUIRED];
}
```

### Repeated Fields (Batch Operations)

```proto
// Batch operations with size limits
repeated string names = 1 [
  (buf.validate.field).repeated.min_items = 1,
  (buf.validate.field).repeated.max_items = 100,
  (buf.validate.field).repeated.items.string.max_len = 200
];

// Labels with pattern and size constraints
repeated string labels = 2 [
  (buf.validate.field).repeated.max_items = 50,
  (buf.validate.field).repeated.items.string = {
    pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
    max_len: 64
  }
];
```

## Special Cases

### Characters vs Bytes

```proto
// Use max_len for Unicode character count (recommended)
string title = 1 [(buf.validate.field).string.max_len = 200];

// Use max_bytes for strict byte limits (rare)
string hash = 2 [(buf.validate.field).string.max_bytes = 64];
```

**When to use `max_bytes`:**

- Binary data (hashes, tokens)
- Protocol-specific limits (e.g., HTTP header size)
- Storage-constrained fields

**Default**: Use `max_len` for all text fields.

### Optional vs Required Fields

```proto
message CreateRequest {
  // Required field - must have min_len
  string title = 1 [
    (google.api.field_behavior) = REQUIRED,
    (buf.validate.field).string.min_len = 1,
    (buf.validate.field).string.max_len = 200
  ];

  // Optional field - no min_len needed
  string description = 2 [(buf.validate.field).string.max_len = 1000];
}
```

### Null vs Empty String

Protobuf doesn't distinguish between null and empty string. Use `min_len = 1` to require non-empty strings.

## Quick Reference

### By Use Case

| I need to validate...    | Use Tier | Use Size | Example                                         |
| ------------------------ | -------- | -------- | ----------------------------------------------- |
| Issue title, group title | SMALL    | 200      | `[(buf.validate.field).string.max_len = 200]`   |
| Group description        | MEDIUM   | 1,000    | `[(buf.validate.field).string.max_len = 1000]`  |
| Issue description        | LARGE    | 10,000   | `[(buf.validate.field).string.max_len = 10000]` |
| Issue comment            | XLARGE   | 65,536   | `[(buf.validate.field).string.max_len = 65536]` |
| Label, tag, version      | TINY     | 64       | `[(buf.validate.field).string.max_len = 64]`    |

### By Tier

| Tier   | Size   | Example Annotation                              |
| ------ | ------ | ----------------------------------------------- |
| TINY   | 64     | `[(buf.validate.field).string.max_len = 64]`    |
| SMALL  | 200    | `[(buf.validate.field).string.max_len = 200]`   |
| MEDIUM | 1,000  | `[(buf.validate.field).string.max_len = 1000]`  |
| LARGE  | 10,000 | `[(buf.validate.field).string.max_len = 10000]` |
| XLARGE | 65,536 | `[(buf.validate.field).string.max_len = 65536]` |

---

## References

- **GitHub API**: [github-limits](https://github.com/dead-claudia/github-limits) - 65,536 char comment limit
- **Google Cloud Platform**: [Resource Manager Constraints](https://cloud.google.com/resource-manager/docs/organization-policy/using-constraints) - 200/1,000/2,000 char limits
- **Kubernetes**: [Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) - 63 char label limit
- **buf.validate**: [Official Documentation](https://buf.build/docs/protovalidate)
- **Protovalidate Rules**: [Standard Rules](https://buf.build/docs/protovalidate/schemas/standard-rules/)
- **Google AIPs**: [API Improvement Proposals](https://google.aip.dev)
- **AIP-133**: [Create Method](https://google.aip.dev/133) - Standard method for creating resources

---

## Version History

- **2025-10-30**:
  - Initial version with 6-tier system (64/200/1K/10K/64K/1M) based on GitHub and GCP standards
  - Added nested resource validation pattern (AIP-133)
  - Clarified that interceptors only validate requests, not responses
