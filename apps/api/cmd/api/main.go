// Command api is the local all-in-one server: the staff and patient routes on one port, for
// `make dev`. On AWS they run as separate functions (cmd/staff-api, cmd/patient-api; D-4).
package main

import (
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
	"github.com/rlamba89/poised/apps/api/internal/serve"
)

func main() { serve.Main("api", httpapi.NewRouter) }
