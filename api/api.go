// Package api embeds the OpenAPI spec so the server can serve it.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPI []byte
