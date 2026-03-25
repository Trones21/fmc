//go:build integration

package main_test

import (
	"os"
	"os/exec"
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

// runFmcExpectFail runs fmc and returns combined output; the test fails if the
// command unexpectedly succeeds.
func runFmcExpectFail(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected fmc to fail but it succeeded\noutput: %s", out)
	}
	return string(out)
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
	assertContains(t, output, "| yes")
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

func TestCreateFrom(t *testing.T) {
	// title: "Quasi-Patterns and Programming Strategies"
	// → slug: quasi-patterns-and-programming-strategies
	dir := copyToTemp(t, "quasi-design-patterns.md")
	runFmc(t, "-createFrom", "title:slug:always:transform:urlsafe", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "quasi-design-patterns.md"))
	assertContains(t, content, "quasi-patterns-and-programming-strategies")
}

func TestCreateFromCopy(t *testing.T) {
	// copy title → display_title without any transform
	dir := copyToTemp(t, "quasi-design-patterns.md")
	runFmc(t, "-createFrom", "title:display_title:always", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "quasi-design-patterns.md"))
	assertContains(t, content, "display_title:")
	assertContains(t, content, "Quasi-Patterns")
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

func TestListEmptyForKey(t *testing.T) {
	// gang-of-four.md has Last_Update: "" and Tags: [""] (non-empty list, won't match)
	// quasi-design-patterns.md has Last_Update: ""
	output := runFmc(t,
		"-listEmptyForKey", "Last_Update",
		"-files", "example-files/gang-of-four.md,example-files/quasi-design-patterns.md",
	)
	assertContains(t, output, "gang-of-four.md")
	assertContains(t, output, "quasi-design-patterns.md")
	assertContains(t, output, "Last_Update")
	// summary should show count of 2
	assertContains(t, output, "| Last_Update | 2")
}

func TestListEmptyWhitespace(t *testing.T) {
	// Create a temp file with a whitespace-only property value
	dir := t.TempDir()
	content := "---\nid: \"ws-test\"\ntitle: \"   \"\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "ws.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	output := runFmc(t, "-listEmptyForKey", "title", "-dir", dir)
	assertContains(t, output, "ws.md")
	assertContains(t, output, "title")
}

func TestListEmptyAll(t *testing.T) {
	dir := t.TempDir()
	// file with two empty props
	c1 := "---\ntitle: \"\"\ndescription: \"\"\nid: \"abc\"\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(c1), 0644); err != nil {
		t.Fatal(err)
	}
	// file with one empty prop
	c2 := "---\ntitle: \"Hello\"\ndescription: \"\"\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte(c2), 0644); err != nil {
		t.Fatal(err)
	}
	output := runFmc(t, "-listEmpty", "-dir", dir)
	// description is empty in both files — should appear with count 2
	assertContains(t, output, "| description")
	assertContains(t, output, "| 2")
	// title is empty in only one file
	assertContains(t, output, "| title")
	assertContains(t, output, "| 1")
}

func TestListEmptyDetails(t *testing.T) {
	dir := t.TempDir()
	c1 := "---\ntitle: \"\"\ndescription: \"\"\nid: \"abc\"\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(c1), 0644); err != nil {
		t.Fatal(err)
	}
	c2 := "---\ntitle: \"Hello\"\ndescription: \"\"\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte(c2), 0644); err != nil {
		t.Fatal(err)
	}
	output := runFmc(t, "-listEmptyDetails", "-dir", dir)
	// a.md has 2 empty props — should sort first (count desc)
	assertContains(t, output, "a.md")
	assertContains(t, output, "| 2 ")
	assertContains(t, output, "b.md")
	assertContains(t, output, "| 1 ")

	// sort by name
	outputName := runFmc(t, "-listEmptyDetails", "-sortBy", "name", "-dir", dir)
	assertContains(t, outputName, "a.md")
	assertContains(t, outputName, "b.md")
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
	assertContains(t, output, "| ok")
	assertContains(t, output, "out-of-order.md")
	assertContains(t, output, "| out_of_order")
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

// ---------------------------------------------------------------------------
// -genID / -genIDOverwriteInvalid
// ---------------------------------------------------------------------------

func TestGenIDMissing(t *testing.T) {
	// Create a file with no id property.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "noid.md"), []byte("---\ntitle: No ID\n---\nBody.\n"), 0644)
	runFmc(t, "-genID", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "noid.md"))
	assertContains(t, content, "id:")
	// value should look like a UUID (contains dashes in UUID positions)
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "id:") {
			if !strings.Contains(line, "-") {
				t.Errorf("expected UUID value in id line, got: %s", line)
			}
		}
	}
}

func TestGenIDEmpty(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "emptyid.md"), []byte("---\nid: \"\"\ntitle: Empty ID\n---\nBody.\n"), 0644)
	runFmc(t, "-genID", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "emptyid.md"))
	// id should now have a UUID value, not be empty
	assertNotContains(t, content, "id: \"\"")
	assertContains(t, content, "id:")
}

func TestGenIDPreservesExisting(t *testing.T) {
	// gang-of-four.md has id: "oc-1b30b803" — should not be overwritten.
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-genID", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertContains(t, content, `id: "oc-1b30b803"`)
}

func TestGenIDOverwriteInvalid(t *testing.T) {
	// non-uuid-id.md has id: my-doc-1 which is not a UUID.
	dir := copyToTemp(t, "non-uuid-id.md")
	runFmc(t, "-genID", "-genIDOverwriteInvalid", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "non-uuid-id.md"))
	assertNotContains(t, content, "my-doc-1")
	assertContains(t, content, "id:")
}

// ---------------------------------------------------------------------------
// -checkType / -tryCast
// ---------------------------------------------------------------------------

func TestCheckType(t *testing.T) {
	// disable-string.md has disable: "false" (string, not bool).
	output := runFmc(t, "-checkType", "disable:bool", "-files", "example-files/disable-string.md,example-files/disable-bool.md")
	assertContains(t, output, "disable-string.md")
	assertNotContains(t, output, "disable-bool.md")
	assertContains(t, output, "string")
}

func TestCheckTypeConforming(t *testing.T) {
	// disable-bool.md has disable: false (correct bool).
	output := runFmc(t, "-checkType", "disable:bool", "-files", "example-files/disable-bool.md")
	assertContains(t, output, "all files conform")
}

func TestTryCastBool(t *testing.T) {
	// disable-string.md has disable: "false" — should be cast to false (bool).
	dir := copyToTemp(t, "disable-string.md")
	runFmc(t, "-tryCast", "disable:bool", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "disable-string.md"))
	assertContains(t, content, "disable: false")
	assertNotContains(t, content, `disable: "false"`)
}

func TestTryCastInt(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "intprop.md"), []byte("---\nid: \"1\"\norder: \"42\"\n---\nBody.\n"), 0644)
	runFmc(t, "-tryCast", "order:int", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "intprop.md"))
	assertContains(t, content, "order: 42")
	assertNotContains(t, content, `order: "42"`)
}

func TestSetValueStaticTypedBool(t *testing.T) {
	// Using the :bool type specifier should write a real boolean, not a string.
	dir := copyToTemp(t, "gang-of-four.md")
	runFmc(t, "-setValue", "disable:static:false:bool", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "gang-of-four.md"))
	assertContains(t, content, "disable: false")
	assertNotContains(t, content, `disable: "false"`)
}

// ---------------------------------------------------------------------------
// -checkFormat
// ---------------------------------------------------------------------------

func TestCheckFormatDate(t *testing.T) {
	// scalar-date.md has last_update: "20240505" — matches YYYYMMDD.
	// out-of-order.md has last_update: "2024-01-01" — does NOT match YYYYMMDD.
	output := runFmc(t,
		"-checkFormat", "last_update:YYYYMMDD",
		"-files", "example-files/scalar-date.md,example-files/out-of-order.md",
	)
	assertNotContains(t, output, "scalar-date.md")
	assertContains(t, output, "out-of-order.md")
}

func TestCheckFormatUUIDValid(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "good.md"), []byte("---\nid: \"a3f8b2c1-1234-4abc-8def-000000000000\"\n---\nBody.\n"), 0644)
	os.WriteFile(filepath.Join(dir, "bad.md"), []byte("---\nid: my-slug\n---\nBody.\n"), 0644)
	output := runFmc(t, "-checkFormat", "id:uuid", "-dir", dir)
	// only bad.md should appear; good.md should not
	assertContains(t, output, "bad.md")
	// good.md should not appear (value my-slug is the violation marker, not the file name)
	assertContains(t, output, "my-slug")
	assertNotContains(t, output, "a3f8b2c1")
}

// ---------------------------------------------------------------------------
// -keysToTop / -keysToBottom
// ---------------------------------------------------------------------------

func TestKeysToTop(t *testing.T) {
	// out-of-order.md has: title, id, last_update
	// After -keysToTop id, id should appear before title.
	dir := copyToTemp(t, "out-of-order.md")
	runFmc(t, "-keysToTop", "id", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "out-of-order.md"))
	idPos := strings.Index(content, "id:")
	titlePos := strings.Index(content, "title:")
	if idPos >= titlePos {
		t.Errorf("expected id to appear before title after keysToTop\ncontent:\n%s", content)
	}
}

func TestKeysToBottom(t *testing.T) {
	// scalar-date.md has: id, title, last_update
	// After -keysToBottom id, id should appear after last_update.
	dir := copyToTemp(t, "scalar-date.md")
	runFmc(t, "-keysToBottom", "id", "-dir", dir)
	content := readFile(t, filepath.Join(dir, "scalar-date.md"))
	idPos := strings.Index(content, "\nid:")
	lastUpdatePos := strings.Index(content, "last_update:")
	if idPos <= lastUpdatePos {
		t.Errorf("expected id to appear after last_update after keysToBottom\ncontent:\n%s", content)
	}
}

func TestKeysToTopMissingKeyNotified(t *testing.T) {
	// Requesting a key that doesn't exist should mention it in the plan, not fail.
	output := runFmc(t, "-keysToTop", "nonexistent_key", "-files", "example-files/scalar-date.md")
	assertContains(t, output, "nonexistent_key")
}

// ---------------------------------------------------------------------------
// -analyzeSEO
// ---------------------------------------------------------------------------

func TestAnalyzeSEO(t *testing.T) {
	// seo-partial.md has title/slug/image but is missing description and keywords.
	// draft-doc.md has draft: true → should be excluded.
	output := runFmc(t,
		"-analyzeSEO", "-plugin", "docs",
		"-files", "example-files/seo-partial.md,example-files/draft-doc.md",
	)
	assertContains(t, output, "Total Files: 2")
	assertContains(t, output, "Unlisted or Draft Files: 1")
	assertContains(t, output, "SEO Analyzed Files: 1")
	assertContains(t, output, "description")
	assertContains(t, output, "keywords")
}

func TestAnalyzeSEORequiresPlugin(t *testing.T) {
	cmd := runFmcExpectFail(t, "-analyzeSEO", "-files", "example-files/seo-partial.md")
	assertContains(t, cmd, "-plugin")
}

// ---------------------------------------------------------------------------
// -listValues
// ---------------------------------------------------------------------------

func TestListValues(t *testing.T) {
	// scalar-date.md and date-iso.md both have last_update but different values.
	// out-of-order.md also has last_update: "2024-01-01" — same as date-iso.md.
	output := runFmc(t,
		"-listValues", "last_update",
		"-files", "example-files/scalar-date.md,example-files/date-iso.md,example-files/out-of-order.md",
	)
	assertContains(t, output, `Values for "last_update"`)
	assertContains(t, output, "20240505")
	assertContains(t, output, "2024-05-05")
	assertContains(t, output, "2024-01-01")
}

func TestListValuesMissingProperty(t *testing.T) {
	// gang-of-four.md has no "title" in lower-case; it's missing from template props.
	output := runFmc(t,
		"-listValues", "title",
		"-files", "example-files/gang-of-four.md,example-files/scalar-date.md",
	)
	assertContains(t, output, "(missing)")
}

// ---------------------------------------------------------------------------
// -listDateFormats / -listDateFormatsDetail
// ---------------------------------------------------------------------------

func TestListDateFormats(t *testing.T) {
	output := runFmc(t,
		"-listDateFormats", "last_update",
		"-files", "example-files/scalar-date.md,example-files/date-iso.md,example-files/out-of-order.md",
	)
	assertContains(t, output, `Date formats for "last_update"`)
	assertContains(t, output, "YYYYMMDD")
	assertContains(t, output, "YYYY-MM-DD")
}

func TestListDateFormatsUnrecognized(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad-date.md"), []byte("---\nid: \"bd-1\"\nlast_update: \"05/05/24\"\n---\nBody.\n"), 0644)
	output := runFmc(t, "-listDateFormats", "last_update", "-dir", dir)
	assertContains(t, output, "(unrecognized)")
	assertContains(t, output, "Unrecognized values by length")
	assertContains(t, output, "8 chars")
	assertContains(t, output, "Tip:")
}

func TestListDateFormatsDetail(t *testing.T) {
	output := runFmc(t,
		"-listDateFormatsDetail", "last_update",
		"-files", "example-files/scalar-date.md,example-files/date-iso.md",
	)
	assertContains(t, output, `Date format detail for "last_update"`)
	assertContains(t, output, "scalar-date.md")
	assertContains(t, output, "YYYYMMDD")
	assertContains(t, output, "date-iso.md")
	assertContains(t, output, "YYYY-MM-DD")
	// table has pipe separators and correct filenames
	assertContains(t, output, "| ")
	assertContains(t, output, "scalar-date.md")
}

// ---------------------------------------------------------------------------
// -analyze
// ---------------------------------------------------------------------------

func TestAnalyze(t *testing.T) {
	output := runFmc(t,
		"-t", "example-files/template.json",
		"-analyze",
		"-files", "example-files/gang-of-four.md,example-files/scalar-date.md,example-files/zabout.md",
	)
	// header row
	assertContains(t, output, "File")
	assertContains(t, output, "Placement")
	assertContains(t, output, "Missing Props")
	// summary
	assertContains(t, output, "Files analyzed: 3")
	assertContains(t, output, "Missing front matter")
	assertContains(t, output, "Missing properties from template")
}

func TestAnalyzeIssuesOnly(t *testing.T) {
	output := runFmc(t,
		"-t", "example-files/template.json",
		"-analyze",
		"-issues-only",
		"-files", "example-files/scalar-date.md,example-files/gang-of-four.md",
	)
	// scalar-date.md is clean vs template — should be suppressed with -issues-only
	// gang-of-four.md is missing title — should appear
	assertContains(t, output, "gang-of-four.md")
}

// ── generateSources / rollup ────────────────────────────────────────────────

func TestGenerateSourcesFilepath(t *testing.T) {
	// Build a small tree: docs/technical/go/tutorial.md
	dir := t.TempDir()
	subdir := filepath.Join(dir, "technical", "go")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: \"Tutorial\"\ntags: []\nkeywords: []\n---\nBody.\n"
	filePath := filepath.Join(subdir, "tutorial.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	runFmc(t, "-generateSources", "filepath", "-files", filePath)

	got := readFile(t, filePath)
	assertContains(t, got, "tag_sources:")
	assertContains(t, got, "filepath:")
	assertContains(t, got, "tag_list:")
	assertContains(t, got, "technical")
	assertContains(t, got, "go")
	assertContains(t, got, "keyword_sources:")
	assertContains(t, got, "keyword_list:")
	assertContains(t, got, "date_last_generated:")
}

func TestGenerateSourcesUnknown(t *testing.T) {
	dir := copyToTemp(t, "scalar-date.md")
	runFmcExpectFail(t, "-generateSources", "llm.unknown", "-dir", dir)
}

func TestRollupTags(t *testing.T) {
	dir := t.TempDir()
	content := `---
tags:
  - existing
tag_sources:
  filepath:
    date_last_generated: "2024-01-01"
    tag_list:
      - technical
      - go
---
Body.
`
	filePath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	runFmc(t, "-rollup", "tags", "-rollupSources", "filepath", "-files", filePath)

	got := readFile(t, filePath)
	// existing tag preserved + new tags added
	assertContains(t, got, "existing")
	assertContains(t, got, "technical")
	assertContains(t, got, "go")
}

func TestRollupNoPreserve(t *testing.T) {
	dir := t.TempDir()
	content := `---
tags:
  - old-tag
tag_sources:
  filepath:
    date_last_generated: "2024-01-01"
    tag_list:
      - new-tag
---
Body.
`
	filePath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	runFmc(t, "-rollup", "tags", "-rollupSources", "filepath", "-rollupNoPreserve", "-files", filePath)

	got := readFile(t, filePath)
	assertContains(t, got, "new-tag")
	assertNotContains(t, got, "old-tag")
}

func TestRollupAll(t *testing.T) {
	dir := t.TempDir()
	content := `---
tags: []
tag_sources:
  filepath:
    date_last_generated: "2024-01-01"
    tag_list:
      - go
  llm:
    gpt-4o:
      date_last_generated: "2024-01-01"
      tag_list:
        - tutorial
---
Body.
`
	filePath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	runFmc(t, "-rollup", "tags", "-rollupSources", "all", "-files", filePath)

	got := readFile(t, filePath)
	assertContains(t, got, "go")
	assertContains(t, got, "tutorial")
}

func TestRollupTagsAndKeywords(t *testing.T) {
	dir := t.TempDir()
	content := `---
tags: []
keywords: []
tag_sources:
  filepath:
    date_last_generated: "2024-01-01"
    tag_list:
      - go
keyword_sources:
  filepath:
    date_last_generated: "2024-01-01"
    keyword_list:
      - golang
---
Body.
`
	filePath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	runFmc(t, "-rollup", "tags,keywords", "-rollupSources", "filepath", "-files", filePath)

	got := readFile(t, filePath)
	assertContains(t, got, "go")
	assertContains(t, got, "golang")
}

func TestRollupMissingSourcesFlag(t *testing.T) {
	dir := copyToTemp(t, "scalar-date.md")
	runFmcExpectFail(t, "-rollup", "tags", "-dir", dir)
}
