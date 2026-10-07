//go:build integration

package apitest_test

import (
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

func TestMain(m *testing.M) { apitest.Main(m) }
