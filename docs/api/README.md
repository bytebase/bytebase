# api.bytebase.com

Static [Scalar](https://scalar.com/) site for the OpenAPI spec generated at
`proto/gen/grpc-doc/openapi.yaml`.

`.github/workflows/deploy-api-docs.yml` copies this directory plus the spec into
`out/` and deploys it to the Cloudflare Pages project `bytebase-api` on every
push to `main` that touches either. The custom domain is attached in the
Cloudflare dashboard.

Preview locally:

```bash
mkdir -p /tmp/api-docs && cp docs/api/* proto/gen/grpc-doc/openapi.yaml /tmp/api-docs && python3 -m http.server -d /tmp/api-docs 8080
```
