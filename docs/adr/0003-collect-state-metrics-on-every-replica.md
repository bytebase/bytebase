# Collect state metrics on every replica

Every Bytebase replica serves installation-scoped state metrics from `/metrics` and computes them synchronously when scraped. This avoids leader election, failover gaps, and additional scrape topology at the cost of duplicate computation across the small replica set; add caching only when a metric-specific benchmark shows it is necessary.
