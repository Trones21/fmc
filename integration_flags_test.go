//go:build integration

package main_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// copyToTemp copies the named files from example-files/ into a fresh temp
// directory and returns the directory path. The directory is removed via
// t.Cleanup automatically.
func copyToTemp(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join("example-files", f))
		if err != nil {
			t.Fatalf("copyToTemp: read %s: %v", f, err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), data, 0644); err != nil {
			t.Fatalf("copyToTemp: write %s: %v", f, err)
		}
	}
	return dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected to contain %q\ngot:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected NOT to contain %q\ngot:\n%s", substr, s)
	}
}

// ---------------------------------------------------------------------------
// Read-only / analysis flags
// ---------------------------------------------------------------------------

func TestListExtraProps(t *testing.T) {
	// gang-of-four.md and quasi-design-patterns.md both have Last_Update and
	// Tags which are not in template.json.
	output := runFmc(t, "-t", "example-files/template.json", "-listExtraProps", "-dir", "example-files")
	assertContains(t, output, "Last_Update")
	assertContains(t, output, "Tags")
}

func TestListMissingProps(t *testing.T) {
	// gang-of-four.md is missing "title" (and lowercase last_update/tags).
	output := runFmc(t, "-t", "example-files/template.json", "-listMissingProps", "-dir", "example-files")
	assertContains(t, output, "gang-of-four.md")
	assertContains(t, output, "title")
}

func TestInspectProp(t *testing.T) {
	// nested-date.md has last_update: { date: "20240505" }
	// inspecting last_update should reveal "date" as a sub-property.
	output := runFmc(t, "-inspectProp", "last_update", "-files", "example-files/nested-date.md")
	assertContains(t, output, "yes")
	assertContains(t, output, "date")
}

func TestInspectPropScalar(t *testing.T) {
	// scalar-date.md has last_update: "20240505" — no sub-properties.
	output := runFmc(t, "-inspectProp", "last_update", "-files", "example-files/scalar-date.md")
	assertContains(t, output, "yes")
	// IsScalar: depth column should show "-"
	assertContains(t, output, "| yes | - |")
}

func TestKeepNVPS(t *testing.T) {
	// With keepNVPS=0 the directory prefix should be hidden.
	output := runFmc(t,
		"-inspectProp", "id",
		"-keepNVPS", "0",
		"-files", "example-files/gang-of-four.md",
	)
	assertContains(t, output, "<hidden>/gang-of-four.md")
	assertNotContains(t, output, "example-files/gang-of-four.md")
}

// ---------------------------------------------------------------------------
// Mutation flags (each test copies fixtures into a temp dir)
// ---------------------------------------------------------------------------

func TestReplaceKey(t *testing.T) {
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-replaceKey", "Last_Update:last_update", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertContains(t, content, "last_update:")
	assertNotContains(t, content, "Last_Update:")
}

func TestCreateSlug(t *testing.T) {
	// title: "Quasi-Patterns and Programming Strategies"
	// → slug: quasi-patterns-and-programming-strategies
	dir := copyToTemp(t, "quasi-design-patterns.md")
	runFmc(t, "-createSlug", "title:slug", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "quasi-design-patterns.md"))
	assertContains(t, content, "quasi-patterns-and-programming-strategies")
}

func TestSetValueStatic(t *testing.T) {
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-setValue", "status:static:draft", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertContains(t, content, "status: draft")
}

func TestSetValueComputedToday(t *testing.T) {
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-setValue", "last_update:computed:today:always", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertContains(t, content, time.Now().Format("2006-01-02"))
}

func TestSetValueTransformNest(t *testing.T) {
	// scalar-date.md: last_update: "20240505"
	// After nest: last_update: { date: "20240505" }
	dir := copyToTemp(t, "scalar-date.md")
	runFmc(t, "-setValue", "last_update.date:transform:copy:last_update:always", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "scalar-date.md"))
	assertContains(t, content, "date:")
	assertContains(t, content, "20240505")
}

func TestSetValueTransformLift(t *testing.T) {
	// nested-date.md: last_update: { date: "20240505" }
	// After lift: last_update: "20240505" (map replaced by scalar, .date gone)
	dir := copyToTemp(t, "nested-date.md")
	runFmc(t, "-setValue", "last_update:transform:copy:last_update.date:always", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "nested-date.md"))
	// last_update should be a scalar (value on same line, not a block map)
	assertContains(t, content, "last_update: ")
	assertContains(t, content, "20240505")
	// the indented "date:" sub-key should be gone
	assertNotContains(t, content, "\n    date:")
}

func TestListEmpty(t *testing.T) {
	// gang-of-four.md has Last_Update: "" and Tags: [""] (non-empty list, won't match)
	// quasi-design-patterns.md has Last_Update: ""
	output := runFmc(t,
		"-listEmpty", "Last_Update",
		"-files", "example-files/gang-of-four.md,example-files/quasi-design-patterns.md",
	)
	assertContains(t, output, "gang-of-four.md")
	assertContains(t, output, "quasi-design-patterns.md")
	assertContains(t, output, "Last_Update")
	// summary should show count of 2
	assertContains(t, output, "| Last_Update | 2 |")
}

func TestListEmptyWhitespace(t *testing.T) {
	// Create a temp file with a whitespace-only property value
	dir := t.TempDir()
	content := "---\nid: \"ws-test\"\ntitle: \"   \"\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "ws.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	output := runFmc(t, "-listEmpty", "title", "-dir", dir)
	assertContains(t, output, "ws.md")
	assertContains(t, output, "title")
}

func TestRemoveEmpty(t *testing.T) {
	// gang-of-four.md has Last_Update: "" — should be deleted.
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-removeEmpty", "Last_Update", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertNotContains(t, content, "Last_Update:")
}

func TestAddMissingProps(t *testing.T) {
	// gang-of-four.md is missing "title" from template.json.
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-t", "example-files/template.json", "-addMissingProps", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertContains(t, content, "title:")
}

func TestAnalyzeOrder(t *testing.T) {
	// scalar-date.md: id, title, last_update  → matches order-template.json order → ok
	// out-of-order.md: title, id, last_update → does not match              → out_of_order
	output := runFmc(t,
		"-t", "example-files/order-template.json",
		"-analyzeOrder",
		"-files", "example-files/scalar-date.md,example-files/out-of-order.md",
	)
	assertContains(t, output, "scalar-date.md")
	assertContains(t, output, "| ok |")
	assertContains(t, output, "out-of-order.md")
	assertContains(t, output, "| out_of_order |")
	assertContains(t, output, "1 in order")
	assertContains(t, output, "1 out of order")
}

func TestAnalyzeOrderIssuesOnly(t *testing.T) {
	output := runFmc(t,
		"-t", "example-files/order-template.json",
		"-analyzeOrder",
		"-issues-only",
		"-files", "example-files/scalar-date.md,example-files/out-of-order.md",
	)
	// in-order file should be suppressed
	assertNotContains(t, output, "scalar-date.md")
	assertContains(t, output, "out-of-order.md")
}

func TestAnalyzeOrderExcluded(t *testing.T) {
	// zabout.md has no front matter → excluded
	output := runFmc(t,
		"-t", "example-files/order-template.json",
		"-analyzeOrder",
		"-files", "example-files/zabout.md,example-files/scalar-date.md",
	)
	assertContains(t, output, "1 excluded")
}

func TestCreateFrontMatter(t *testing.T) {
	// zabout.md has no front matter; template.json defines id, title, last_update, tags.
	dir := copyToTemp(t, "zabout.md")
	runFmc(t,
		"-t", "example-files/template.json",
		"-createFrontMatter",
		"-fmDefault", "title:About",
		"-dir", dir,
	)
	content := readFile(t, filepath.Join(dir, "zabout.md"))
	assertContains(t, content, "---")
	assertContains(t, content, "id:")
	assertContains(t, content, "title: About")
	// original body content should still be present
	assertContains(t, content, "PK_ToDo")
}

func TestCreateFrontMatterSkipsExisting(t *testing.T) {
	// All files in example-files that have front matter should be skipped,
	// leaving only zabout.md as a candidate — but we run without zabout.md.
	dir := copyToTemp(t, "gang-of-four.md", "quasi-design-patterns.md")
	output := runFmc(t,
		"-t", "example-files/template.json",
		"-createFrontMatter",
		"-dir", dir,
	)
	assertContains(t, output, "No files need front matter creation.")
}

func TestRemoveExtraProps(t *testing.T) {
	// gang-of-four.md has Last_Update and Tags which are not in template.json.
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-t", "example-files/template.json", "-removeExtraProps", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertNotContains(t, content, "Last_Update:")
	assertNotContains(t, content, "Tags:")
}
