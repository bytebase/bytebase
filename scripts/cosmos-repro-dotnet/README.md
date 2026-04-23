# cosmos-repro-dotnet — BYT-9239 reproduction harness

Reproduces the Cosmos DB query failures described in BYT-9239 using the
Microsoft .NET SDK (`Microsoft.Azure.Cosmos`). The goal is to determine
whether the failures are:

- **Go-SDK-specific** (Bytebase's forked `azcosmos`) → queries pass here, fail in Go
- **Gateway/server-level** → queries fail in both SDKs

## Prerequisites

- .NET 9 SDK (`brew install dotnet`)
- Azure CLI, logged in to the subscription holding `bytebase-cosmostest`

## Run

```bash
export COSMOS_ENDPOINT="https://bytebase-cosmostest.documents.azure.com:443/"
export COSMOS_KEY="$(az cosmosdb keys list \
  --name bytebase-cosmostest \
  --resource-group rg-bytebase-cosmostest \
  --type keys --query primaryMasterKey -o tsv)"
export COSMOS_DB="testdb"
export COSMOS_CONTAINER="WorldCities"

# Gateway mode (same as Bytebase's default)
dotnet run --project . -- --mode gateway --verbose | tee out-gateway.md

# Direct mode (TCP, bypasses gateway)
dotnet run --project . -- --mode direct --verbose | tee out-direct.md
```

Each run prints a markdown table of per-query status: PASS / FAIL, rows returned,
elapsed milliseconds, and a truncated error.

## Scope

The test list covers the query groups marked FAIL or PARTIAL in the PDF:
`Basic SELECT TOP`, `ORDER BY`, `Aggregation` (aliased + `VALUE`),
`GROUP BY`, `DISTINCT`, `VALUE COUNT`, `OFFSET/LIMIT`. It also includes
one known-passing query (`WHERE c.country = "AE"`) as a sanity check.

Container partition key is `/country` — queries without a `country` filter
are cross-partition and are the ones expected to stress-test the SDK's
cross-partition orchestration.
