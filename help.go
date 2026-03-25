package main

import (
	"flag"
	"fmt"
	"os"
)

// printFlag prints a single flag entry using its registered usage string.
// valueHint is shown after the flag name (e.g. "<path>", "<value>"); pass ""
// for boolean flags.
func printFlag(out *os.File, name, valueHint string) {
	f := flag.Lookup(name)
	if f == nil {
		return
	}
	const col = 26
	label := "-" + name
	if valueHint != "" {
		label += " " + valueHint
	}
	if len(label) > col {
		fmt.Fprintf(out, "  %s\n  %-*s %s\n", label, col, "", f.Usage)
	} else {
		fmt.Fprintf(out, "  %-*s %s\n", col, label, f.Usage)
	}
}

func section(out *os.File, title string) {
	fmt.Fprintf(out, "\n%s\n", title)
}

func printHelp() {
	out := os.Stderr
	fmt.Fprintln(out, "Usage: fmc [flags]")

	section(out, "Front Matter Template:")
	printFlag(out, "template", "<path>")
	printFlag(out, "t", "<path>")

	section(out, "Files to Operate On:")
	printFlag(out, "dir", "<path>")
	printFlag(out, "files", "<file1,file2>")

	section(out, "List / Analyze  (read-only, no writes):")
	printFlag(out, "placementAudit", "")
	printFlag(out, "listExtraProps", "")
	printFlag(out, "listMissingProps", "")
	printFlag(out, "listEmpty", "")
	printFlag(out, "listEmptyDetails", "")
	printFlag(out, "listEmptyForKey", "<propertyName>")
	printFlag(out, "sortBy", "<name|count>")
	printFlag(out, "checkFormat", "<key:FORMAT>")
	printFlag(out, "checkType", "<key:type>")
	printFlag(out, "listValues", "<propertyName>")
	printFlag(out, "listDateFormats", "<propertyName>")
	printFlag(out, "listDateFormatsDetail", "<propertyName>")
	printFlag(out, "analyze", "")
	printFlag(out, "analyzeOrder", "")
	printFlag(out, "analyzeSEO", "")
	printFlag(out, "plugin", "<docs|blog>")
	printFlag(out, "inspectProp", "<key>")
	printFlag(out, "issues-only", "")
	printFlag(out, "verbose", "")

	section(out, "Make Changes — Single Property:")
	printFlag(out, "setValue", "<key:source:value[:action]>")
	printFlag(out, "replaceKey", "<OldKey:NewKey>")
	printFlag(out, "createFrom", "<from:to[:action][:transform:fn]>")
	printFlag(out, "genID", "")
	printFlag(out, "genIDOverwriteInvalid", "")
	printFlag(out, "tryCast", "<key:type>")
	printFlag(out, "removeEmpty", "<propertyName>")

	section(out, "Make Changes — Multi Property:")
	printFlag(out, "createFrontMatter", "")
	printFlag(out, "onManualReview", "")
	printFlag(out, "fmDefault", "<key:value>")
	printFlag(out, "keysToTop", "<key1,key2,...>")
	printFlag(out, "keysToBottom", "<key1,key2,...>")
	printFlag(out, "addMissingProps", "")
	printFlag(out, "removeExtraProps", "")
	printFlag(out, "allProps", "")
	printFlag(out, "fullConform", "")
	printFlag(out, "fixOrder", "")

	section(out, "Tags & Keywords — Source Generation:")
	printFlag(out, "generateSources", "<filepath>")

	section(out, "Tags & Keywords — Rollup:")
	printFlag(out, "rollup", "<tags|keywords|tags,keywords>")
	printFlag(out, "rollupSources", "<source1,source2|all>")
	printFlag(out, "rollupNoPreserve", "")

	section(out, "Display Options:")
	printFlag(out, "keepNonVariadicPathSegments", "<N>")
	printFlag(out, "keepNVPS", "<N>")

	section(out, "Other:")
	printFlag(out, "help", "")
	printFlag(out, "examples", "")

	fmt.Fprintln(out, "\nRun 'fmc -examples' for usage examples.")
	fmt.Fprintln(out, "Run 'fmc help <flag>' for detailed help on a specific flag.")
	fmt.Fprintln(out, "Run 'fmc commonWorkflows' for common multi-step cleanup sequences.")
}

func printExamples() {
	fmt.Print(`Examples:
  Audit front matter placement:
    fmc -dir ./docs -placementAudit

  Run all checks (placement, missing, extra, empty, order):
    fmc -t template.json -analyze -dir ./docs

  Find extra/misspelled keys across a directory:
    fmc -t template.json -dir ./docs -listExtraProps

  Summary of empty properties across all keys:
    fmc -listEmpty -dir ./docs

  Per-file empty-property breakdown:
    fmc -listEmptyDetails -sortBy name -dir ./docs

  List files where a specific property is empty:
    fmc -listEmptyForKey description -dir ./docs

  Check a date property conforms to a format:
    fmc -checkFormat "last_update.date:YYYYMMDD" -dir ./docs

  Check a property is the correct type:
    fmc -checkType "disable:bool" -dir ./docs

  Cast a property to the correct type:
    fmc -tryCast "disable:bool" -dir ./docs

  Analyze SEO front matter (Docusaurus docs plugin):
    fmc -analyzeSEO -plugin docs -dir ./docs

  Add missing template keys (empty value):
    fmc -t template.json -addMissingProps -dir ./docs

  Remove keys not in the template:
    fmc -t template.json -removeExtraProps -dir ./docs

  Set a value (static, computed, or transform):
    fmc -setValue "last_update:computed:today:if_empty" -dir ./docs

  Generate UUIDs for missing id fields:
    fmc -genID -dir ./docs

  Add front matter to files that are missing it:
    fmc -t template.json -createFrontMatter -dir ./docs

  Add front matter to manual-review files specifically:
    fmc -t template.json -createFrontMatter -onManualReview -dir ./docs

  Move id and title to the top, tags to the bottom:
    fmc -keysToTop id,title -keysToBottom tags,last_update -dir ./docs

  Policy subcommand help:
    fmc policy help
    fmc policy list-functions

  Flag-specific help:
    fmc help setValue
    fmc help addMissingProps
    fmc help removeExtraProps
    fmc help createFrom
    fmc help replaceKey
    fmc help createFrontMatter
    fmc help inspectProp
    fmc help listEmpty
    fmc help checkFormat
    fmc help analyzeSEO
    fmc help analyzeOrder
`)
}

func runHelpTopic(topic string) {
	switch topic {
	case "createFrom":
		fmt.Print(`-createFrom from:to[:action][:transform:fn]

  Derives the value of one front matter key from another, writing the result
  to a destination key. An optional transform controls how the value is
  produced; without one the source value is copied as-is.

Actions:
  (none)          add_if_missing — only set if the destination key is absent (default)
  if_empty        set if the destination is absent or ""
  always          always overwrite the destination

Transforms:
  (none)          copy — copy the source value unchanged
  transform:copy       same as no transform
  transform:urlsafe    URL-safe slug (lowercase, spaces→dashes, special chars stripped)
  transform:slug       alias for urlsafe

Examples:
  Copy title → display_title if missing:
    fmc -createFrom title:display_title -dir ./docs

  Build a URL-safe slug from title, only if slug is empty:
    fmc -createFrom title:slug:if_empty:transform:urlsafe -dir ./docs

  Always regenerate slug from title:
    fmc -createFrom title:slug:always:transform:urlsafe -dir ./docs

  Multiple derivations in one pass:
    fmc -createFrom title:slug:if_empty:transform:urlsafe -createFrom name:id_slug:always:transform:urlsafe -dir ./docs

`)
	case "replaceKey":
		fmt.Print(`-replaceKey OldKey:NewKey

  Renames a front matter key while keeping its value. The old key is removed
  and the value is written to the new key. Useful for fixing typos or casing.

Examples:
  Rename a single key:
    fmc -replaceKey Last_Updated:last_update -dir ./docs

  Rename multiple keys in one pass:
    fmc -replaceKey Last_Updated:last_update -replaceKey Disable:is_disabled -dir ./docs

`)
	case "setValue":
		fmt.Print(`-setValue key:source:value[:action]

  Sets a front matter property value. The source determines how the value is
  produced. Action controls when the write happens.

Sources:
  static     Use the literal value (optionally suffixed with a type: bool, string, int, float)
  computed   Run a built-in deterministic function (today, uuid, path_segments)
  transform  Derive a value from another property (supports dotted paths); requires fn:from_key
  llm        Run an AI function (describe, tags, title) — requires API key

Actions:
  (none)     add_if_missing — only set if the key is absent (default)
  if_empty   overwrite_if_empty — set if the key is absent or ""
  always     overwrite_always — always overwrite

  Note: action is detected by suffix. Values ending in ":always" or ":if_empty"
  are treated as action markers, so literal values with those suffixes are not
  supported directly.

Examples:
  Add a static draft status if missing:
    fmc -setValue "status:static:draft" -dir ./docs

  Set a boolean value (write false not "false"):
    fmc -setValue "disable:static:false:bool:always" -dir ./docs

  Always stamp last_update with today's date:
    fmc -setValue "last_update:computed:today:always" -dir ./docs

  Add a UUID only if id is missing:
    fmc -setValue "id:computed:uuid" -dir ./docs

  Multiple values in one pass:
    fmc -setValue "status:static:draft" -setValue "last_update:computed:today:if_empty" -dir ./docs

  Nest a scalar into a child key (last_update: "20240505" → last_update.date: "20240505"):
    fmc -setValue "last_update.date:transform:copy:last_update:always" -dir ./docs

  Lift a child key back to the parent (last_update.date: "20240505" → last_update: "20240505"):
    fmc -setValue "last_update:transform:copy:last_update.date:always" -dir ./docs

`)
	case "createFrontMatter":
		fmt.Print(`-createFrontMatter  (requires -t)

  Adds a front matter block to every file that currently has none. Only keys
  defined in the template are written. Use -fmDefault to supply initial values;
  any key without a default is written with an empty value.

  Before writing, shows each target file with its first 5 lines so you can
  verify the right files are being targeted.

-fmDefault key:value  (repeatable)

  Supplies a default value for a template key during -createFrontMatter.
  Keys not covered by -fmDefault receive an empty value.

Examples:
  Add front matter with all-empty values:
    fmc -t template.json -createFrontMatter -dir ./docs

  Add front matter with some defaults pre-filled:
    fmc -t template.json -createFrontMatter \
        -fmDefault "title:Untitled" \
        -fmDefault "status:draft" \
        -dir ./docs

`)
	case "inspectProp":
		fmt.Print(`-inspectProp <key>

  Inspects the nested YAML structure of a named property across all files.
  For each file, shows whether the property is present, its maximum depth, and
  any sub-properties found. Ends with an aggregated summary table.

  Repeatable — pass multiple times to inspect several properties in one pass.

Examples:
  Inspect the "tags" property across a directory:
    fmc -inspectProp tags -dir ./docs

  Inspect multiple properties at once:
    fmc -inspectProp tags -inspectProp metadata -dir ./docs

`)
	case "addMissingProps":
		fmt.Print(`-addMissingProps

  Adds any template keys that are absent from a file's front matter. The new
  keys are written with an empty value. Requires -t to specify the template.

Examples:
  Add missing keys across a directory:
    fmc -t template.json -addMissingProps -dir ./docs

  Add missing keys to a single file:
    fmc -t template.json -addMissingProps -files ./docs/my-post.md

`)
	case "analyzeOrder":
		fmt.Print(`-analyzeOrder  (requires -t)

  Checks whether each file's front matter keys appear in the same order as the
  template. Files that are missing one or more template properties are excluded
  from the check (they cannot be fairly compared). The summary shows how many
  files were excluded.

  Respects -issues-only (suppress files that are in order) and
  -keepNonVariadicPathSegments / -keepNVPS for path display.

Examples:
  Check key order across a directory:
    fmc -t template.json -analyzeOrder -dir ./docs

  Show only out-of-order files:
    fmc -t template.json -analyzeOrder -issues-only -dir ./docs

`)
	case "listEmpty":
		fmt.Print(`-listEmpty

  Scans every key in every file and shows a ranked table of which properties
  are most frequently empty (nil, "", or whitespace-only) across the whole set.

-listEmptyDetails  (sortable with -sortBy name|count)

  Per-file breakdown showing each file that has at least one empty property,
  the count of empty properties, and the list of empty keys. Default sort is
  by count descending; use -sortBy name for alphabetical by filename.

-listEmptyForKey <propertyName>  (repeatable)

  Lists every file where the named property exists in the front matter but its
  value is empty. Files where the property is absent are not reported — use
  -listMissingProps for that. Pass the flag multiple times to check several
  properties in one pass.

Examples:
  Ranked empty-property summary across all keys:
    fmc -listEmpty -dir ./docs

  Per-file breakdown sorted by count (default):
    fmc -listEmptyDetails -dir ./docs

  Per-file breakdown sorted by filename:
    fmc -listEmptyDetails -sortBy name -dir ./docs

  Find files with an empty description:
    fmc -listEmptyForKey description -dir ./docs

  Check multiple properties at once:
    fmc -listEmptyForKey description -listEmptyForKey tags -dir ./docs

`)
	case "checkFormat":
		fmt.Print(`-checkFormat key:FORMAT  (repeatable)

  Lists files where the named property is present but its string value does not
  parse as the given date format. Properties that are absent are not reported.
  Dotted paths are supported (e.g. last_update.date).

Named formats:
  uuid   RFC 4122 UUID (e.g. a3f8b2c1-1234-4abc-8def-000000000000)

Date format tokens:
  YYYY   four-digit year
  MM     two-digit month (01–12)
  DD     two-digit day (01–31)
  HH     two-digit hour (00–23)
  mm     two-digit minute (00–59)
  ss     two-digit second (00–59)

Examples:
  Check id is a valid UUID:
    fmc -checkFormat "id:uuid" -dir ./docs

  Check last_update.date is YYYYMMDD:
    fmc -checkFormat "last_update.date:YYYYMMDD" -dir ./docs

  Check multiple properties:
    fmc -checkFormat "id:uuid" -checkFormat "last_update.date:YYYYMMDD" -dir ./docs

`)
	case "analyzeSEO":
		fmt.Print(`-analyzeSEO  (requires -plugin)

  Reports how many files are missing or have empty values for SEO-relevant
  front matter properties. Files where draft or unlisted is true are excluded
  from the analysis.

  The header shows:
    Total Files          — all files passed to fmc
    Unlisted or Draft    — files skipped due to draft/unlisted
    SEO Analyzed Files   — files actually checked

-plugin <docs|blog>

  Selects the Docusaurus plugin whose SEO properties are checked.

  docs  title, description, keywords, image, slug
  blog  title, title_meta, description, keywords, image, slug

Examples:
  Analyze SEO coverage for the docs plugin:
    fmc -analyzeSEO -plugin docs -dir ./docs

  Analyze a blog directory:
    fmc -analyzeSEO -plugin blog -dir ./blog

`)
	case "removeExtraProps":
		fmt.Print(`-removeExtraProps

  Removes any front matter keys that are not defined in the template. Shows a
  preview of all deletions before writing. Requires -t to specify the template.

Examples:
  Remove extra keys across a directory:
    fmc -t template.json -removeExtraProps -dir ./docs

  Remove extra keys from a single file:
    fmc -t template.json -removeExtraProps -files ./docs/my-post.md

`)
	case "generateSources", "rollup":
		fmt.Print(`Tags & Keywords — Source Generation and Rollup
==============================================

Front matter structure managed by these flags:

  tags: [go, tutorial]           # Docusaurus public /tags/ navigation
  keywords: [golang, beginner]   # SEO <meta keywords>
  internal_tags: [needs-review]  # Never surfaced to Docusaurus

  tag_sources:
    filepath:
      date_last_generated: "2024-01-01"
      tag_list: [technical, go]
    llm:
      gpt-4o:
        date_last_generated: "2024-01-01"
        tag_list: [tutorial, beginners]

  keyword_sources:
    filepath:
      date_last_generated: "2024-01-01"
      keyword_list: [technical, go]
    llm:
      gpt-4o:
        date_last_generated: "2024-01-01"
        keyword_list: [golang, api]

-generateSources <source>

  Populates tag_sources.<source> and keyword_sources.<source> for every file.
  Sets date_last_generated to today. Overwrites any previous run from that
  source. Currently supported sources:

    filepath   Derives segments from the file's directory path
               (drops root prefix and filename; inner dirs become tags)

-rollup <tags|keywords|tags,keywords>
-rollupSources <source1,source2|all>
-rollupNoPreserve

  Merges staged source lists into the top-level tags or keywords field.
  -rollupSources selects which sources to include; use 'all' to include every
  source present. Nested LLM sources use dot notation (e.g. llm.gpt-4o).
  By default existing values are preserved (set union). Pass -rollupNoPreserve
  to replace existing values with only the union of the selected sources —
  removed items are shown explicitly in the preview.

Examples:
  Generate filepath sources for all docs:
    fmc -generateSources filepath -dir ./docs

  Roll up filepath tags into the tags field (preserve existing):
    fmc -rollup tags -rollupSources filepath -dir ./docs

  Roll up all sources into both tags and keywords:
    fmc -rollup tags,keywords -rollupSources all -dir ./docs

  Replace tags entirely with what llm.gpt-4o suggests:
    fmc -rollup tags -rollupSources llm.gpt-4o -rollupNoPreserve -dir ./docs

`)
	default:
		fmt.Printf("no help topic %q\n\n", topic)
		fmt.Println("Available help topics:")
		fmt.Println("  fmc help setValue")
		fmt.Println("  fmc help addMissingProps")
		fmt.Println("  fmc help removeExtraProps")
		fmt.Println("  fmc help createFrom")
		fmt.Println("  fmc help replaceKey")
		fmt.Println("  fmc help createFrontMatter")
		fmt.Println("  fmc help inspectProp")
		fmt.Println("  fmc help listEmpty")
		fmt.Println("  fmc help checkFormat")
		fmt.Println("  fmc help analyzeSEO")
		fmt.Println("  fmc help analyzeOrder")
		fmt.Println("  fmc help generateSources")
		fmt.Println()
		fmt.Println("For the full flag list run: fmc help")
		os.Exit(1)
	}
}

func printCommonWorkflows() {
	fmt.Print(`Common Workflows
================

These are multi-step sequences you can run during front matter cleanup.
Each step is a separate fmc command — they are not composed automatically.

──────────────────────────────────────────────────────────────────────
Workflow: Find empty properties, then remove them
──────────────────────────────────────────────────────────────────────

Step 1 — See which properties are empty across your files:

  fmc -listEmpty -dir ./docs

  This shows a ranked table of which keys have the most empty values,
  so you can decide what to clean up.

Step 2a — Remove a specific set of empty keys:

  fmc -removeEmpty "description,last_update" -dir ./docs

Step 2b — Remove ALL empty keys across every file:

  fmc -removeEmpty all -dir ./docs

  Use the per-file breakdown first if you want to review before bulk-deleting:

  fmc -listEmptyDetails -sortBy name -dir ./docs

`)
}

func runPolicyCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: fmc policy <command>")
		fmt.Println("Commands:")
		fmt.Println("  help             Show policy file format and actions")
		fmt.Println("  list-functions   List all available computed and transform functions")
		return
	}

	switch args[0] {
	case "help":
		fmt.Print(`Policy file format (JSON):

  {
    "<key>": { "action": "<action>", "source": "<source>", "fn": "<fn>", "from": "<key>", "params": {} }
  }

Actions:
  add_if_missing      Add the key if absent
  overwrite_always    Always set the value
  overwrite_if_empty  Set only if missing or empty
  preserve            Leave untouched (default)
  rename_from         Rename an old key to this one; requires "from"

Sources:
  static              Use the literal "value" field
  computed            Run a built-in function (see list-functions)
  transform           Derive a value from another property; requires "from"

`)
	case "list-functions":
		fmt.Println("Computed functions  (\"source\": \"computed\")")
		fmt.Println()
		fmt.Println("  today            Current date as YYYY-MM-DD")
		fmt.Println("  uuid             Random UUID v4")
		fmt.Println("  path_segments    Segments from the file path added to the tags property")
		fmt.Println("                   Drops the first and last segment (root prefix and filename)")
		fmt.Println("                   Params:")
		fmt.Println("                     skip  (int)  Drop an additional N leading segments (default 0)")
		fmt.Println()
		fmt.Println("Transform functions  (\"source\": \"transform\")  — require \"from\": \"<key>\"  (dotted paths supported)")
		fmt.Println()
		fmt.Println("  copy             Copy the source value as-is (useful for nest/lift with dotted keys)")
		fmt.Println("  slug             URL-safe slug (lowercase, spaces→dashes, special chars stripped)")
	default:
		fmt.Printf("unknown policy command %q\n", args[0])
		fmt.Println("Run 'fmc policy' to see available commands.")
		os.Exit(1)
	}
}
