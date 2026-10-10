package modules

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const modulePrefix = "selfemi/internal/modules/"

var schemaRef = regexp.MustCompile(`(?i)\b(?:from|join|into|update|table)\s+([a-z_][a-z0-9_]*)\.`)

type violation struct {
	file   string
	detail string
}

func checkModule(t *testing.T, root, module string) []violation {
	t.Helper()
	var found []violation
	dir := filepath.Join(root, module)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, src, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if strings.HasPrefix(p, modulePrefix) && !strings.HasPrefix(p, modulePrefix+module) {
				found = append(found, violation{path, "imports " + p})
			}
		}
		for _, m := range schemaRef.FindAllStringSubmatch(string(src), -1) {
			if schema := strings.ToLower(m[1]); schema != module {
				found = append(found, violation{path, "uses schema " + schema})
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", module, err)
	}
	return found
}

func moduleNames(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestModulesStayInsideTheirBoundary(t *testing.T) {
	names := moduleNames(t, ".")
	if len(names) < 2 {
		t.Fatalf("expected at least two modules, found %v", names)
	}
	for _, module := range names {
		for _, v := range checkModule(t, ".", module) {
			t.Errorf("module %s: %s %s", module, v.file, v.detail)
		}
	}
}

func TestBoundaryCheckCatchesViolations(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha/a.go", "package alpha\nimport _ \"selfemi/internal/modules/beta\"\nconst q = `SELECT * FROM beta.things`\n")
	write("beta/b.go", "package beta\nconst q = `SELECT * FROM beta.things JOIN beta.other ON true`\n")

	if got := checkModule(t, root, "alpha"); len(got) != 2 {
		t.Fatalf("alpha violations = %v, want an import and a schema violation", got)
	}
	if got := checkModule(t, root, "beta"); len(got) != 0 {
		t.Fatalf("beta violations = %v, want none", got)
	}
}
