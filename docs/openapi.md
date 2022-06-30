# Bytebase OpenAPI

Bytebase OpenAPI implements the OpenAPI specification with Swagger.

## Dev

### Required tools

* [`swag`](https://github.com/swaggo/swag) for the Swagger command line.
* [`echo-swagger`](https://github.com/swaggo/echo-swagger) Swagger middleware for echo.

### Comment for OpenAPIs

1. Add general API annotations in `./server/server.go`.
2. Add API operation annotations in your controller code.

> Note: you need to comment for a specific controller function, otherwise the swagger cannot extract the comment as docs. For example:

```go
// Swagger doc (cannot work)
// @Summary  Health check for service
// @Accept  */*
// @Produce  plain
// @Router  /healthz [get]
e.GET("/healthz", func(c echo.Context) error {
    return c.String(http.StatusOK, "OK!\n")
})

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
swag init -g ./server.go -d ./server --output docs/openapi --parseDependency
```

This should generate new docs under `./docs/openapi` folder based on your comments.

### Initial the Swagger

Update code in `./server/server.go`:

```go
import (
    echoSwagger "github.com/swaggo/echo-swagger" // The Swagger middleware for echo
    _ "github.com/bytebase/bytebase/docs/openapi" // Generate by Swagger
)

// Add Swagger Router
e.GET("/swagger/*", echoSwagger.WrapHandler)
```

Then start the service, you can visit the Swagger dashboard on `/swagger/index.html`
