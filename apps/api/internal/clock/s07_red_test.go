package clock

import "testing"

// Deliberately failing (S07, plan F4 Checks): a push with a failing unit test must stop the
// pipeline at Synth, before anything is deployed. The next commit reverts this.
func TestS07PipelineStopsOnAFailingTest(t *testing.T) {
	t.Fatal("deliberately failing (S07): the pipeline must stop here, before any deploy")
}
