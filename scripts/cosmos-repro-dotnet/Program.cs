using System.Diagnostics;
using Microsoft.Azure.Cosmos;

// Reproduction harness for BYT-9239.
// Runs the queries that failed with the Bytebase (Go) Cosmos driver against
// the same Azure account using the Microsoft .NET SDK, to decide whether the
// failures are SDK-side (Go fork) or gateway/server-side.
//
// Required environment variables:
//   COSMOS_ENDPOINT   e.g. https://bytebase-cosmostest.documents.azure.com:443/
//   COSMOS_KEY        primary master key
//   COSMOS_DB         database name (default: testdb)
//   COSMOS_CONTAINER  container name (default: WorldCities)
//
// Flags:
//   --mode gateway|direct   (default: gateway) — matches the two modes we want to test
//   --verbose               print per-query detail in addition to the summary table

static class Program
{
    static async Task<int> Main(string[] args)
    {
        var mode = ParseFlag(args, "--mode", "gateway").ToLowerInvariant();
        var verbose = args.Contains("--verbose");

        var endpoint = Env("COSMOS_ENDPOINT");
        var key      = Env("COSMOS_KEY");
        var db       = Env("COSMOS_DB", "testdb");
        var coll     = Env("COSMOS_CONTAINER", "WorldCities");

        var cmode = mode switch
        {
            "gateway" => ConnectionMode.Gateway,
            "direct"  => ConnectionMode.Direct,
            _ => throw new ArgumentException($"unknown --mode {mode}")
        };

        Console.WriteLine($"# .NET SDK reproduction — mode={mode} db={db} coll={coll}");
        Console.WriteLine($"SDK: Microsoft.Azure.Cosmos (Microsoft.Azure.Cosmos assembly)");
        Console.WriteLine();

        var opts = new CosmosClientOptions
        {
            ConnectionMode = cmode,
            ApplicationName = "bytebase-byt9239-repro",
        };
        using var client = new CosmosClient(endpoint, key, opts);
        var container = client.GetContainer(db, coll);

        var tests = Tests();
        var rows = new List<Result>();
        foreach (var t in tests)
        {
            var r = await RunOne(container, t);
            rows.Add(r);
            if (verbose)
            {
                Console.WriteLine($"[{r.Status}] {r.Id}  rows={r.RowCount}  elapsed={r.ElapsedMs}ms");
                if (r.Error != null) Console.WriteLine($"    ERR: {Short(r.Error, 260)}");
            }
        }

        // Summary markdown table
        Console.WriteLine();
        Console.WriteLine($"## Results ({mode} mode)");
        Console.WriteLine();
        Console.WriteLine("| # | Query | Status | Rows | Elapsed (ms) | Error (truncated) |");
        Console.WriteLine("|---|-------|--------|------|-------------:|-------------------|");
        foreach (var r in rows)
        {
            var err = r.Error == null ? "" : Short(r.Error, 140).Replace("|", "\\|").Replace("\n", " ");
            var q   = r.Sql.Replace("|", "\\|");
            Console.WriteLine($"| {r.Id} | `{q}` | {r.Status} | {r.RowCount} | {r.ElapsedMs} | {err} |");
        }
        return 0;
    }

    record TestCase(string Id, string Sql);
    record Result(string Id, string Sql, string Status, int RowCount, long ElapsedMs, string? Error);

    static IEnumerable<TestCase> Tests() => new[]
    {
        // --- 1. Basic SELECT ---
        new TestCase("1.1",  "SELECT * FROM c"),
        new TestCase("1.2",  "SELECT c.id, c.name, c.country, c.population FROM c"),
        new TestCase("1.3",  "SELECT TOP 10 * FROM c"),

        // --- 2. WHERE ---
        new TestCase("2.1",  "SELECT * FROM c WHERE c.country = \"AE\""),
        new TestCase("2.2",  "SELECT * FROM c WHERE c._ts > 1743702439"),
        new TestCase("2.3",  "SELECT * FROM c WHERE c._ts >= 1743702439"),
        new TestCase("2.4",  "SELECT * FROM c WHERE c.country = \"AE\" OR c.country = \"AD\""),
        new TestCase("2.5",  "SELECT * FROM c WHERE c.country IN (\"AE\", \"US\", \"GB\", \"AD\")"),
        new TestCase("2.6",  "SELECT * FROM c WHERE CONTAINS(c.name, \"Fujayrah\")"),
        new TestCase("2.7",  "SELECT * FROM c WHERE STARTSWITH(c.name, \"Al\")"),
        new TestCase("2.8",  "SELECT * FROM c WHERE c.countryRegion != \"4\""),
        new TestCase("2.9",  "SELECT * FROM c WHERE STRINGTONUMBER(c.population) BETWEEN 10000 AND 100000"),

        // --- 3. Projection & Aliasing ---
        new TestCase("3.1",  "SELECT c.name AS cityName, c.country AS countryCode, c.population AS pop FROM c"),
        new TestCase("3.2",  "SELECT c.name, c.population, STRINGTONUMBER(c.population) / 1000 AS populationInThousands FROM c"),
        new TestCase("3.3",  "SELECT CONCAT(c.name, \" (\", c.country, \")\") AS label FROM c"),

        // --- 4. ORDER BY ---
        new TestCase("4.1",  "SELECT c.name, c.population FROM c ORDER BY c.population ASC"),
        new TestCase("4.2",  "SELECT c.name, c.population FROM c ORDER BY c.population DESC"),
        new TestCase("4.3",  "SELECT c.country, c.name, c.population FROM c ORDER BY c.country ASC, c.population DESC"),

        // --- 5. Aggregation ---
        new TestCase("5.1",  "SELECT COUNT(1) AS totalRecords FROM c"),
        new TestCase("5.1b", "SELECT VALUE COUNT(1) FROM c"),
        new TestCase("5.2",  "SELECT COUNT(1) AS aeCount FROM c WHERE c.country = \"AE\""),
        new TestCase("5.2b", "SELECT VALUE COUNT(1) FROM c WHERE c.country = \"AE\""),
        new TestCase("5.3",  "SELECT SUM(StringToNumber(c.population)) AS totalPop FROM c"),
        new TestCase("5.3b", "SELECT VALUE SUM(StringToNumber(c.population)) FROM c"),
        new TestCase("5.4",  "SELECT AVG(StringToNumber(c.population)) AS avgPop FROM c"),
        new TestCase("5.4b", "SELECT VALUE AVG(StringToNumber(c.population)) FROM c"),
        new TestCase("5.5",  "SELECT MIN(StringToNumber(c.population)) AS minPop, MAX(StringToNumber(c.population)) AS maxPop FROM c WHERE StringToNumber(c.population) > 0"),

        // --- 6. GROUP BY ---
        new TestCase("6.1",  "SELECT c.country, COUNT(1) AS cityCount FROM c GROUP BY c.country"),
        new TestCase("6.2",  "SELECT c.countryRegion, COUNT(1) AS count, SUM(StringToNumber(c.population)) AS totalPop FROM c GROUP BY c.countryRegion"),

        // --- 7. String built-ins ---
        new TestCase("7.1",  "SELECT UPPER(c.name) AS upperName, LOWER(c.country) AS lowerCountry, LENGTH(c.name) AS nameLen FROM c"),

        // --- 8. Math built-ins ---
        new TestCase("8.1",  "SELECT c.name, ROUND(StringToNumber(c.latitude)) AS lat, ROUND(StringToNumber(c.longitude)) AS lon FROM c"),

        // --- 9. Type-checking built-ins ---
        new TestCase("9.1",  "SELECT c.name, IS_STRING(c.name) AS isStr, IS_NUMBER(c.population) AS isNum FROM c"),
        new TestCase("9.2",  "SELECT c.id, IS_DEFINED(c.region) AS hasRegion, IS_NULL(c.code) AS codeNull FROM c"),

        // --- 10. DISTINCT ---
        new TestCase("10.1", "SELECT DISTINCT c.country FROM c"),
        new TestCase("10.2", "SELECT DISTINCT VALUE c.countryRegion FROM c"),

        // --- 11. VALUE keyword ---
        new TestCase("11.1", "SELECT VALUE c.name FROM c WHERE c.country = \"AE\""),
        new TestCase("11.2", "SELECT VALUE COUNT(1) FROM c"),

        // --- 12. OFFSET / LIMIT ---
        new TestCase("12.1", "SELECT * FROM c ORDER BY c.name OFFSET 0 LIMIT 10"),
        new TestCase("12.2", "SELECT * FROM c ORDER BY c.name OFFSET 10 LIMIT 10"),

        // --- 13. Geo-spatial ---
        new TestCase("13.1", "SELECT c.name, ST_DISTANCE({\"type\": \"Point\", \"coordinates\": [StringToNumber(c.longitude), StringToNumber(c.latitude)]}, {\"type\": \"Point\", \"coordinates\": [55.2708, 25.2048]}) AS distFromDubaiInMeters FROM c"),
    };

    static async Task<Result> RunOne(Container c, TestCase t)
    {
        var sw = Stopwatch.StartNew();
        try
        {
            var opt = new QueryRequestOptions { MaxItemCount = 1000 };
            // Cap at a safe upper bound — we just want to know if the query succeeds.
            // For `SELECT * FROM c` the container has thousands of docs; we stop at 10k.
            const int cap = 10_000;

            // Use `dynamic` so VALUE queries that return bare scalars (numbers, strings) deserialize cleanly.
            using var iter = c.GetItemQueryIterator<dynamic>(new QueryDefinition(t.Sql), requestOptions: opt);
            int count = 0;
            while (iter.HasMoreResults)
            {
                var page = await iter.ReadNextAsync();
                count += page.Count;
                if (count >= cap) break;
            }
            sw.Stop();
            return new Result(t.Id, t.Sql, "PASS", count, sw.ElapsedMilliseconds, null);
        }
        catch (Exception e)
        {
            sw.Stop();
            return new Result(t.Id, t.Sql, "FAIL", 0, sw.ElapsedMilliseconds, e.Message);
        }
    }

    static string Env(string name, string? def = null)
    {
        var v = Environment.GetEnvironmentVariable(name);
        if (!string.IsNullOrWhiteSpace(v)) return v!;
        if (def != null) return def;
        throw new InvalidOperationException($"missing env var {name}");
    }

    static string ParseFlag(string[] args, string flag, string def)
    {
        for (int i = 0; i < args.Length - 1; i++)
            if (args[i] == flag) return args[i + 1];
        return def;
    }

    static string Short(string s, int n) => s.Length <= n ? s : s.Substring(0, n) + "…";
}
