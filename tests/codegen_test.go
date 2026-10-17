package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/openapi"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/bootstrap"
)

var generatedTypes = filepath.Join("..", "resources", "js", "generated", "api.ts")

// generatedTS returns what `vukem codegen:types -zod` emits for the API,
// made to compile against the app's zod 4.
//
// workaround(vukem v0.23.0): codegen:types -zod emits zod 3 code (z.ZodTypeDef), so it imports zod's bundled zod/v3.
func generatedTS(t *testing.T) []byte {
	t.Helper()

	app := vukemtest.New(t, bootstrap.New)

	doc, err := app.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}

	ts, err := openapi.TypeScript(doc, true)
	if err != nil {
		t.Fatal(err)
	}

	return []byte(strings.Replace(ts, `import { z } from "zod";`, `import { z } from "zod/v3";`, 1))
}

// The committed TypeScript types and zod schemas match the API. Regenerate
// with: UPDATE_GOLDEN=1 go test ./tests -run TestCodegen_TypesUpToDate
func TestCodegen_TypesUpToDate(t *testing.T) {
	t.Parallel()

	got := generatedTS(t)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(generatedTypes, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(generatedTypes) //nolint:gosec // fixed repository path
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), got) {
		t.Fatalf("%s is stale: run UPDATE_GOLDEN=1 go test ./tests -run TestCodegen_TypesUpToDate", generatedTypes)
	}
}
