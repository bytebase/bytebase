# cosmos-repro-go — BYT-9239 reproduction harness (Go)

Go twin of `scripts/cosmos-repro-dotnet`. Runs the full 40-query target
inventory from BYT-9239 against a live Azure Cosmos account through
Bytebase's forked `azcosmos` SDK (which embeds the pure-Go cross-partition
query engine), and prints a markdown PASS/FAIL table in the same shape as
the .NET harness. Pairing both harnesses on the same container is the
regression guard for the BYT-9239 feature set: as long as the Go matrix
matches the .NET matrix (39/40 PASS; the single FAIL is the multi-key
ORDER BY query that needs a composite index — a server-side constraint
that affects every SDK), the Go engine is at .NET parity for this inventory.

## Prerequisites

- Go toolchain (the repo's current `go.mod` version)
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

cd scripts/cosmos-repro-go
go run . --verbose | tee out-go.md
```

The harness uses `NewCrossPartitionQueryItemsPager`, which engages the
pure-Go query engine automatically when the gateway returns its
cross-partition sentinel error. Queries the gateway can still serve
(e.g. single-partition WHERE filters, projections, built-ins) incur no
extra round-trip — they never leave the gateway path.

## Scope

The query list mirrors the .NET harness's 40-query inventory exactly, in
PDF section order. It includes both the FAIL / PARTIAL groups (TOP, ORDER BY,
aggregates in aliased and VALUE form, GROUP BY, DISTINCT, OFFSET/LIMIT,
scalar VALUE aggregates) and the known-passing groups (filters, projections,
string / math / type-checking built-ins, geo-spatial). Running both groups
keeps the two harnesses symmetric: a divergence between the Go and .NET
columns is the signal that the Go engine regressed on something the .NET
SDK handles.

Container partition key is `/country` — queries without a `country` filter
are cross-partition and are the ones exercising the engine.

## Expected matrix

39 PASS / 1 FAIL, matching the .NET run. The one permanent FAIL is
`SELECT c.country, c.name, c.population FROM c ORDER BY c.country ASC, c.population DESC`
(4.3): the Azure gateway rejects multi-key ORDER BY when the container
lacks a composite index on `(country ASC, population DESC)` — a real
server constraint, not an SDK limitation.
