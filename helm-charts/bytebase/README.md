# Bytebase Helm Chart

Install the web-based schema change and version control tool [Bytebase](https://bytebase.com/).

## Installing

`helm -n <YOUR_NAMESPACE> --set "bytebase.option.port"={PORT} --set "bytebase.option.external-url"={EXTERNAL_URL} --set "bytebase.option.pg"={PGDSN} --set "bytebase.version"={VERSION} install <RELEASE_NAME> helm-charts/bytebase`

## Uninstalling

`helm delete <RELEASE_NAME>`

## TODO
- [ ] Add support for [Litestream](https://litestream.io/guides/kubernetes/).
- [ ] Create GitHub Pages for [Helm chart repo](https://medium.com/@mattiaperi/create-a-public-helm-chart-repository-with-github-pages-49b180dbb417).