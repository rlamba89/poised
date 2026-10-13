// Command patient-api serves only the patient's link, /api/p/… (httpapi.NewPatientRouter).
// On AWS it's the patient-api Lambda (D-4; the adapter comes in S06); locally it's a plain
// server, e.g. `make patient-api`.
package main

import (
	"github.com/rlamba89/poised/apps/api/internal/httpapi"
	"github.com/rlamba89/poised/apps/api/internal/serve"
)

func main() { serve.Main("patient-api", httpapi.NewPatientRouter, false) }
