# SQL Download Goldens

These files are the **wire contract** between Bytebase's backend SQL exporters
(`backend/component/export/{csv,json,sql,xlsx}.go`) and the frontend client-side
download module (`frontend/src/utils/sql-download/formats/*`).

## Regenerating

After any backend serializer change:

```bash
go test ./backend/component/export -run TestDownloadGoldens -update -count=1
```

This rewrites every `.csv` / `.json` / `.sql` / `.xlsx` file in this directory
from the curated fixtures in `backend/component/export/download_goldens_fixtures.go`.

Commit the regenerated files in the same PR as the backend change.

## Verifying (CI)

Without `-update`, the same test asserts backend output still matches the
committed bytes. CI runs this without `-update`; drift fails the test with a
"regenerate goldens" message.
