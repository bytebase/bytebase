# Bytebase OpenAPI

Bytebase OpenAPI implements the OpenAPI specification with Swagger.

## Dev

### Required tools

- [`swag`](https://github.com/swaggo/swag) for the Swagger command line.
- [`echo-swagger`](https://github.com/swaggo/echo-swagger) Swagger middleware for echo.

### Comment for OpenAPIs

1. Add general API annotations in `./backend/server/server.go`.
2. Add API operation annotations in your controller code.

> **Note**
> You need to comment for a specific controller function, otherwise the swagger cannot extract the comment as docs. For example:

❌ Incorrect

```go
// Swagger doc (cannot work)
// @Summary  Health check for service
// @Accept  */*
// @Produce  plain
// @Router  /healthz [get]
e.GET("/healthz", func(c echo.Context) error {
    return c.String(http.StatusOK, "OK!\n")
})
```

✅ Correct

```go
// Swagger doc (this can work)
// @Summary  Health check for service
// @Accept  */*
// @Produce  plain
// @Router  /healthz [get]
func healthCheckController(c echo.Context) error {
    return c.String(http.StatusOK, "OK!\n")
}

e.GET("/healthz", healthCheckController)
```

### Generate Swagger docs

Every time you changed the comments, you need to re-run the following command to re-generate the Swagger docs:

```bash
cd bytebase
# generate swagger doc
swag init -g ./backend/server.go -d ./backend/server --output docs/openapi --parseDependency
```

This should generate new docs under `./docs/openapi` folder based on your comments.
Then start the service, you can visit the Swagger dashboard on `/swagger/index.html`
