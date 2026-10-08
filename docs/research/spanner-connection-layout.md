# Spanner connection form layout research

Research date: September 16, 2026. Sources are official product documentation. Visual observations below come from inspecting each documentation screenshot, not from testing the current applications.

## Findings

| Product | Verified pattern | Relevance to Bytebase |
| --- | --- | --- |
| [DBeaver Spanner](https://dbeaver.com/docs/dbeaver/Database-driver-Google-Cloud-Spanner/) | The [connection screenshot](https://dbeaver.com/docs/dbeaver/images/database/spanner/google-cloud-spanner-connection-main.png) stacks Project, Instance, and Database as separate full-width rows sharing one label column. Authentication and Credentials are dropdowns in a separate group. Driver properties and SSH have separate tabs. | Strong direct precedent for stacking cloud resource identifiers rather than creating two independent label columns. |
| [DataGrip PostgreSQL](https://www.jetbrains.com/help/datagrip/postgresql.html) | The [connection screenshot](https://resources.jetbrains.com/help/img/idea/2026.2/db_connection_details_postgresql_default.png) pairs Host and Port on one row; Host is wider than Port (approximately 5:3 input widths in this screenshot). Authentication uses a dropdown. SSH/SSL and Advanced have separate tabs. | Supports treating network address and port as a coupled row, with unequal widths. Exact Bytebase proportions should follow its existing PostgreSQL form, not copy these pixels. |
| [pgAdmin Server dialog](https://www.pgadmin.org/docs/pgadmin4/latest/server_dialog.html) | The [connection screenshot](https://www.pgadmin.org/static/docs/pgadmin4-9.17-docs/_images/server_connection.png) stacks Host name/address, Port, Maintenance database, and other fields; even Port uses the full input width. Parameters, SSH Tunnel, and Advanced are separate tabs. | Evidence for a coherent single scan path, but also a counterexample: compact paired host/port is not a universal product convention. |
| [DataGrip Google Cloud](https://www.jetbrains.com/help/datagrip/clouds-google-cloud.html) | Documentation separates account authentication from selecting cloud databases and documents proxy configuration under Extended Connection Settings. | Supports separating ordinary connection information from less common network configuration. It does not establish an exact Spanner endpoint layout. |

## Recommendation (design inference)

Use the existing Bytebase horizontal form pattern: one shared left label column, Project ID and Instance ID on separate rows, then a long Endpoint field with a compact Port at its right, matching PostgreSQL's Hostname/Port row. Keep Authentication as the existing dropdown, including a sole option, per the stated visual-consistency preference. Align helper text beneath its input region.

This combines the most relevant external precedent (DBeaver's stacked Spanner identifiers) with the local consistency requirement. Grouping identifiers under top labels can save a row, but would introduce another layout grammar within the same form. Changing every connection field to top labels is coherent in isolation but requires a broader cross-engine redesign.

The products' separate network/advanced areas are inspiration for a future optional-endpoint disclosure, not evidence that these products hide precisely the same setting. An existing nondefault endpoint should remain visible when editing.

## Limits

- No usability testing was performed; the recommendation is a design judgment based on the supplied Bytebase screenshot and observed product patterns.
- DBeaver's Spanner text table refers to a Cloud SQL-style instance format; this research uses its screenshot only for layout and does not rely on that questionable format description.
- DataGrip documentation lists Spanner with basic support, but this research did not find a verified current Spanner-specific connection screenshot. Its PostgreSQL screenshot is the source for host/port observations.
