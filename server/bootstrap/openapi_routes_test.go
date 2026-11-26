package bootstrap

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"amiya-eden/config"
	"amiya-eden/global"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestStartupV1RoutesMatchOpenAPI(t *testing.T) {
	oldConfig, oldLogger, oldMode := global.Config, global.CurrentLogger(), gin.Mode()
	t.Cleanup(func() {
		global.Config = oldConfig
		global.SetLogger(oldLogger)
		gin.SetMode(oldMode)
	})
	global.Config = &config.Config{}
	config.ApplyDefaults(global.Config)
	global.Config.Server.Mode = gin.TestMode
	global.SetLogger(zap.NewNop())

	// Go through the same initialization as main. No handler is invoked, so
	// this does not need an HTTP listener, a task service, Redis, or a database.
	const base = "/api/v1"
	actual := make(map[string]struct{})
	for _, route := range InitRouter(nil).Routes() {
		if route.Path != base && !strings.HasPrefix(route.Path, base+"/") {
			continue
		}
		segments := strings.Split(route.Path, "/")
		for i, segment := range segments {
			if name, ok := strings.CutPrefix(segment, ":"); ok {
				segments[i] = "{" + name + "}"
			}
		}
		actual[route.Method+" "+strings.Join(segments, "/")] = struct{}{}
	}

	// Read the source contract rather than the generated Go route table.
	root := filepath.Join("..", "..", "contracts")
	modules, err := filepath.Glob(filepath.Join(root, "modules", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := make(map[string]struct{})
	for _, filename := range append([]string{filepath.Join(root, "openapi.yaml")}, modules...) {
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Paths map[string]map[string]yaml.Node `yaml:"paths"`
		}
		if err := yaml.Unmarshal(data, &spec); err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		for path, item := range spec.Paths {
			for method := range item {
				switch method {
				case "get", "put", "post", "delete", "options", "head", "patch", "trace":
					key := strings.ToUpper(method) + " " + base + path
					if _, duplicate := want[key]; duplicate {
						t.Fatalf("duplicate OpenAPI operation in %s: %s", filename, key)
					}
					want[key] = struct{}{}
				}
			}
		}
	}

	var missing, extra []string
	for key := range want {
		if _, exists := actual[key]; !exists {
			missing = append(missing, key)
		}
	}
	for key := range actual {
		if _, exists := want[key]; !exists {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	t.Logf("OpenAPI operations: %d; registered /api/v1 routes: %d", len(want), len(actual))
	if len(missing) != 0 {
		t.Errorf("present in OpenAPI but not registered:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) != 0 {
		t.Errorf("registered but absent from OpenAPI:\n  %s", strings.Join(extra, "\n  "))
	}
}
