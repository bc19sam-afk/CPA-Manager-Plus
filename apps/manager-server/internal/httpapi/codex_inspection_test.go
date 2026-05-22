package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/seakee/cpa-manager-plus/apps/manager-server/internal/store"
	"github.com/seakee/cpa-manager-plus/apps/manager-server/internal/testutil"
)

func TestCodexInspectionRoutesAreMounted(t *testing.T) {
	cfg := testutil.NewConfig(t)
	handler, db := newCompatHandler(t, cfg, nil)
	managerCfg := store.ManagerConfig{
		CPAConnection: store.ManagerCPAConnectionConfig{
			CPABaseURL:    "http://cpa.local",
			ManagementKey: "management-key",
		},
		Collector: store.ManagerCollectorConfig{
			CollectorMode:  "auto",
			Queue:          "usage",
			PopSide:        "right",
			BatchSize:      100,
			PollIntervalMS: 500,
			QueryLimit:     50000,
		},
		CodexInspection: store.DefaultCodexInspectionConfig(),
	}
	if err := db.SaveManagerConfig(context.Background(), managerCfg); err != nil {
		t.Fatalf("save manager config: %v", err)
	}

	rr := testutil.Request(t, handler, http.MethodGet, "/v0/management/codex-inspection/runs", "", testutil.AdminKey)
	testutil.RequireStatus(t, rr, http.StatusOK)
	if !strings.Contains(rr.Body.String(), `"items"`) {
		t.Fatalf("runs body = %s", rr.Body.String())
	}
}
