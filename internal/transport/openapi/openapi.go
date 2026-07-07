// Package openapi embeds and serves the API spec — the source of truth for
// generated clients (yaver-web) and partner SDKs.
package openapi

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
