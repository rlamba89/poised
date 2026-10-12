// Command staff-api serves only the staff routes (httpapi.NewStaffRouter). On AWS it's the
// staff-api Lambda (D-4; the adapter comes in S06); locally it's a plain server, e.g.
// `make staff-api`.
package main

import (
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
	"github.com/rlamba89/poised/apps/api/internal/serve"
)

func main() { serve.Main("staff-api", httpapi.NewStaffRouter) }
