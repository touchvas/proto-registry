// Package openapi holds the OpenAPI (Swagger 2) file of each api-gateway-service profile,
// which the gateway serves as its docs. `make openapi` writes them.
package openapi

import _ "embed"

// Player documents the player gateway: the player apps' API.
//
//go:embed player.swagger.json
var Player []byte

// Admin documents the admin gateway: the back office's API.
//
//go:embed admin.swagger.json
var Admin []byte
