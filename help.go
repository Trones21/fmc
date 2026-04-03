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

func subsection(out *os.File, title string) {
	fmt.Fprintf(out, "\n  (%s)\n", title)
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
	printFlag(out, "listLength", "")
	printFlag(out, "checkFormat", "<key:FORMAT>")
	printFlag(out, "checkType", "<key:type>")
	printFlag(out, "listValues", "<propertyName>")
	printFlag(out, "listDateFormats", "<propertyName>")
	printFlag(out, "listDateFormatsDetail", "<propertyName>")
	printFlag(out, "analyze", "")
	printFlag(out, "analyzeOrder", "")
	printFlag(out, "analyzeSEO", "")
	printFlag(out, "inspectProp", "<key>")

	section(out, "Make Changes — Single Property:")
	printFlag(out, "setValue", "<key:source:value[:action]>")
	printFlag(out, "replaceKey", "<OldKey:NewKey>")
	printFlag(out, "createFrom", "<from:to[:action][:transform:fn]>")
	printFlag(out, "genID", "")
	printFlag(out, "genIDOverwriteInvalid", "")
	printFlag(out, "tryCast", "<key:type>")

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
	printFlag(out, "removeEmpty", "<propertyName>")
	printFlag(out, "pruneFMIfLinesBelowN", "<N>")
	printFlag(out, "pruneFMIfCharsBelowN", "<N>")
	printFlag(out, "pruneFMKeepProps", "<key1,key2>")

	section(out, "Tags & Keywords — Source Generation:")
	printFlag(out, "generateSources", "<filepath>")

	section(out, "Tags & Keywords — Rollup:")
	printFlag(out, "rollup", "<tags|keywords|tags,keywords>")
	printFlag(out, "rollupSources", "<source1,source2|all>")
	printFlag(out, "rollupNoPreserve", "")

	section(out, "Tags & Keywords — LLM Generation (requires ~/.fmc/config.json):")
	printFlag(out, "generateSources", "<llm.gpt-4o>")
	printFlag(out, "llmFields", "<title,description,tags,keywords>")
	printFlag(out, "llmSkipFresherThan", "<N>")
	printFlag(out, "llmRegenerateIfNewer", "")
	printFlag(out, "llmSkipIfContentLinesBelowN", "<N>")
	printFlag(out, "llmSkipIfContentCharsBelowN", "<N>")
	printFlag(out, "llmSkipIfPropEquals", "<key:value>")

	section(out, "Apply LLM-Generated Values:")
	printFlag(out, "applyLLMGeneratedTitle", "<llm.gpt-4o[:action]>")
	printFlag(out, "applyLLMGeneratedDescription", "<llm.gpt-4o[:action]>")

	section(out, "Output & Behavior Modifiers:")
	subsection(out, "all operations — file filtering")
	printFlag(out, "skipIfContentLinesBelowN", "<N>")
	printFlag(out, "skipIfContentLinesAboveN", "<N>")
	printFlag(out, "skipIfContentCharsBelowN", "<N>")
	printFlag(out, "skipIfContentCharsAboveN", "<N>")
	subsection(out, "-setValue — apply only to files matching content condition")
	printFlag(out, "setValueIfContentLinesBelowN", "<N>")
	printFlag(out, "setValueIfContentLinesAboveN", "<N>")
	printFlag(out, "setValueIfContentCharsBelowN", "<N>")
	printFlag(out, "setValueIfContentCharsAboveN", "<N>")
	subsection(out, "all operations — path display")
	printFlag(out, "keepNonVariadicPathSegments", "<N>")
	printFlag(out, "keepNVPS", "<N>")
	subsection(out, "listEmptyDetails, listLength — sort order")
	printFlag(out, "sortBy", "<key[:desc]>")
	subsection(out, "analyze, analyzeOrder — output filtering")
	printFlag(out, "issues-only", "")
	printFlag(out, "verbose", "")
	subsection(out, "analyzeSEO — required companion")
	printFlag(out, "plugin", "<docs|blog>")

	section(out, "Links:")
	printFlag(out, "extractLinks", "<all|internal|external|images>")
	printFlag(out, "makeLinksAbsolute", "<https://example.com>")
	printFlag(out, "makeLinksRelative", "<https://example.com>")

	section(out, "Export:")
	printFlag(out, "exportJSON", "<output.json>")
	printFlag(out, "urlStartsAfter", "<path>")
	printFlag(out, "exportJSONLinkKey", "<slug|slug_strict|id|filename>")
	printFlag(out, "exportJSONFields", "<id,title,tags,...>")
	printFlag(out, "exportJSONContentLength", "")
	printFlag(out, "exportJSONOnMissing", "<skip_file|include_file_add_empty>")

	section(out, "Other:")
	printFlag(out, "help", "")
	printFlag(out, "examples", "")

	fmt.Fprintln(out, "\nRun 'fmc -examples' for usage examples.")
	fmt.Fprintln(out, "Run 'fmc help <topic>' for detailed help on a specific flag or topic.")
	fmt.Fprintln(out, "Run 'fmc help list' to see all available help topics.")
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
	case "llm":
		fmt.Print(`LLM Source Generation (OpenAI / ChatGPT)
=========================================

Config file: ~/.fmc/config.json
  {
    "openai": {
      "api_key": "sk-...",
      "model":   "gpt-4o"
    },
    "llm": {
      "content_date_field":  "last_update.date",
      "content_date_format": "YYYY-MM-DD"
    }
  }

Run fmc -llmTest to test the OpenAI connection using the API key in ~/.fmc/config.json

Supported models: gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-3.5-turbo

All generated values are STAGED, not written directly to front matter.
Use -applyLLMGeneratedTitle / -applyLLMGeneratedDescription to apply single-
value fields, and -rollup to apply tags/keywords. See: fmc help generateSources

-generateSources llm.<model>

  Sends each file's markdown content to the OpenAI API and stages results into:
    title_sources.llm.<model>.value
    description_sources.llm.<model>.value
    tag_sources.llm.<model>.tag_list
    keyword_sources.llm.<model>.keyword_list
  Each also gets a date_last_generated field set to today (YYYY-MM-DD).

-llmFields <title,description,tags,keywords>

  CSV of fields to generate. Defaults to all four. Omit fields you don't need
  to reduce API cost and latency.

-llmSkipFresherThan <N>

  Skip a file if its date_last_generated for this source is within N days.
  0 (default) means always regenerate.

-llmRegenerateIfNewer

  When used with -llmSkipFresherThan, overrides the skip if the content date
  field (llm.content_date_field in config) is newer than date_last_generated.
  Files missing the content date field are warned and skipped.

-applyLLMGeneratedTitle <source[:action]>
-applyLLMGeneratedDescription <source[:action]>

  Write the staged value from title_sources or description_sources to the
  top-level 'title' or 'description' key. Action controls when to write:
    (none)         add_if_missing — only write if the key is absent (default)
    if_empty       write if absent or empty string
    always         always overwrite

Examples:
  Generate all four fields for every doc:
    fmc -generateSources llm.gpt-4o -dir ./docs

  Generate only tags and keywords:
    fmc -generateSources llm.gpt-4o -llmFields tags,keywords -dir ./docs

  Regenerate only files updated since last LLM run:
    fmc -generateSources llm.gpt-4o -llmSkipFresherThan 7 -llmRegenerateIfNewer -dir ./docs

  Apply staged title if the title field is currently empty:
    fmc -applyLLMGeneratedTitle llm.gpt-4o:if_empty -dir ./docs

  Roll up LLM tags into the tags field (preserving existing):
    fmc -rollup tags -rollupSources llm.gpt-4o -dir ./docs

`)
	case "list":
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
		fmt.Println("  fmc help llm")
		fmt.Println("  fmc help exportJSON")
		fmt.Println("  fmc help links")
		fmt.Println()
		fmt.Println("Run 'fmc commonWorkflows' for common multi-step cleanup sequences.")
		fmt.Println("Run 'fmc policy help' for policy file format.")
		fmt.Println("Run 'fmc policy list-functions' for built-in functions.")

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
	case "links", "extractLinks", "makeLinksAbsolute", "makeLinksRelative":
		fmt.Print(`Links — Extraction and Conversion
=================================

fmc can find every [text](url) link in your markdown body, categorise it, and
write the results back into front matter. It can also bulk-convert links between
relative and absolute forms.

─── Extraction ────────────────────────────────────────────────────────────────

-extractLinks <mode>

  Scans the body of each file for [text](url) markdown links and writes them
  into front matter properties. Always overwrites any existing values.

  Modes:
    all        Extract internal_links, external_links, and image_links
    internal   Extract internal_links only
    external   Extract external_links only
    images     Extract image_links only (![text](url) syntax)

Front matter structure written
------------------------------
  internal_links:
    absolute:   [ /technical/intro, /about ]       # start with /
    relative:   [ ../sibling, ./child/page ]        # no leading /
    anchor:     [ #section-heading ]                # same-document anchors
                                                    # cross-doc anchors go in
                                                    # relative or absolute
  external_links: [ https://github.com, ... ]
  image_links:    [ /img/logo.png, https://cdn/x.png ]

  Links with both a path and an anchor (e.g. /technical/intro#usage) are
  placed in absolute or relative based on the path prefix, not in anchor.

Preview output
--------------
  For each file fmc prints one line per found link:

    New link   — shows up to 40 chars of surrounding context:
      [internal_links.relative] found new (line 12):
        See also [related guide](../pandas/loc-guide) for details.
        → ../pandas/loc-guide

    Existing   — already in front matter, no context printed:
      [external_links] found existing: https://github.com/foo/bar

    Stale      — in front matter but no longer in the document:
      [external_links] link not found in document, will be removed: https://old.example.com

Examples:
  Extract all link types:
    fmc -extractLinks all -dir ./docs

  Extract only external links from a single file:
    fmc -extractLinks external -file ./docs/intro.md

  Extract images only:
    fmc -extractLinks images -dir ./docs

─── Conversion ────────────────────────────────────────────────────────────────

-makeLinksAbsolute <prefix>

  Prepends <prefix> to every internal absolute link (/...) in the file body.
  Relative links (../, ./), anchors (#), and external links are left alone.

  Example:
    Before:  [Intro](/technical/intro)
    Command: fmc -makeLinksAbsolute https://thomasrones.com -dir ./docs
    After:   [Intro](https://thomasrones.com/technical/intro)

-makeLinksRelative <prefix>

  Strips <prefix> from any link that starts with it, leaving the path portion.
  Trailing slash on the prefix is ignored. The resulting relative link always
  starts with /.

  Example:
    Before:  [Intro](https://thomasrones.com/technical/intro)
    Command: fmc -makeLinksRelative https://thomasrones.com -dir ./docs
    After:   [Intro](/technical/intro)

  Both flags operate only on [text](url) syntax — raw URLs in prose and
  <a href> tags are not touched.

`)
	case "exportJSON":
		fmt.Print(`-exportJSON <output.json>

  Exports front matter data for all matched files into a single JSON array.
  Each element contains the front matter fields plus two synthetic fields:
  filepath (the file's path on disk) and link (its URL path).

Field set
---------
  No template (-t)   →  id, title, filepath, link
  With -t             →  all keys in the template + filepath + link

  Note: filepath and link are always synthetic — they are never read from
  front matter, even if those keys exist in the template.

Flags
-----
  -exportJSON <path>
    Output file path. Overwritten without warning if it already exists.

  -urlStartsAfter <segment>
    Filesystem prefix to strip when computing the link URL. Everything up to
    and including this segment is removed, leaving its children as the URL path.

    Example: -urlStartsAfter /home/user/repo/docs
      /home/user/repo/docs/technical/intro.md  →  /technical/intro

  -exportJSONLinkKey <slug|slug_strict|id|filename>   (default: slug)
    Which value to use as the URL path segment:

    slug         Use the slug front matter field. If slug is a relative value
                 (no leading /), it is resolved against the file's directory.
                 Falls back to filename if slug is absent.
    slug_strict  Same as slug but leaves link empty ("") when slug is absent.
                 Use this when you want to identify files that still need a slug.
    id           Use the id field. Falls back to filename if absent.
    filename     Always derive from the file path (ignores slug/id).

  -exportJSONFields <csv>   (optional)
    Explicit comma-separated list of front matter fields to include.
    Takes priority over -t and the built-in default (id, title).
    Example: -exportJSONFields "id,title,tags,keywords"

  -exportJSONContentLength
    Add content_lines and content_chars to each row. Both exclude front matter.
    Useful for filtering stub files downstream.

  -exportJSONOnMissing <skip_file|include_file_add_empty>   (default: skip_file)
    What to do when a file is missing one or more required fields:

    skip_file              Exclude the file from the output. Prints a warning
                           to the console for each skipped file.
    include_file_add_empty Include the file, filling missing fields with "".
                           Prints a warning for each affected file.

Examples
--------
  Minimal export (id + title + link) from a single directory:
    fmc -exportJSON out.json -dir ./docs

  Build an index across multiple subdirectories (pass -dir once per dir):
    fmc -exportJSON site-index.json \
        -dir ./technical \
        -dir ./blog \
        -dir ./about \
        -urlStartsAfter /home/user/repo/docs

  Full export using template fields, with URL prefix stripped:
    fmc -exportJSON out.json -t template.json \
        -dir ./docs \
        -urlStartsAfter /home/user/repo/docs

  Include files even when fields are missing (add empty strings):
    fmc -exportJSON out.json -dir ./docs \
        -exportJSONOnMissing include_file_add_empty

  Use filename instead of slug for the link field:
    fmc -exportJSON out.json -dir ./docs \
        -exportJSONLinkKey filename \
        -urlStartsAfter /home/user/repo/docs

`)
	default:
		fmt.Printf("no help topic %q\n\n", topic)
		fmt.Println("Run 'fmc help list' to see all available help topics.")
		fmt.Println("Run 'fmc help' for the full flag list.")
		os.Exit(1)
	}
}

// workflowEntry is a registered workflow with a short description.
type workflowEntry struct {
	name        string
	description string
	run         func()
}

var workflows = []workflowEntry{
	{
		name:        "cleanEmpty",
		description: "Find empty front matter properties and remove them",
		run:         workflowCleanEmpty,
	},
	{
		name:        "llmGenerate",
		description: "Generate and apply LLM-suggested title, description, tags, and keywords",
		run:         workflowLLMGenerate,
	},
	{
		name:        "addFrontMatter",
		description: "Find files missing front matter and add it with useful defaults",
		run:         workflowAddFrontMatter,
	},
	{
		name:        "fixNesting",
		description: "Fix an accidentally over-nested property by collapsing an unwanted intermediate level",
		run:         workflowFixNesting,
	},
}

func printWorkflowIndex() {
	fmt.Print(`Common Workflows
================

Workflows are multi-step sequences for common front matter tasks.
Each step is a separate fmc command — they are not composed automatically.
Run a workflow by name to see the full step-by-step guide.

Usage:
  fmc commonWorkflows <name>

Available workflows:
`)
	for _, w := range workflows {
		fmt.Printf("  %-20s %s\n", w.name, w.description)
	}
	fmt.Println()
}

func runWorkflow(name string) {
	for _, w := range workflows {
		if w.name == name {
			w.run()
			return
		}
	}
	fmt.Fprintf(os.Stderr, "error: unknown workflow %q\n\n", name)
	fmt.Fprintln(os.Stderr, "Run 'fmc commonWorkflows' to see available workflows.")
	os.Exit(1)
}

func workflowCleanEmpty() {
	fmt.Print(`Workflow: cleanEmpty — Find empty properties, then remove them
==============================================================

These are multi-step sequences you can run during front matter cleanup.
Each step is a separate fmc command — they are not composed automatically.

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

func workflowFixNesting() {
	fmt.Print(`Workflow: fixNesting — Collapse an accidentally over-nested property
====================================================================

Problem
-------
A property ended up with an extra intermediate level that shouldn't be there.
For example, instead of:

  last_update:
    date: 20250618

the file contains:

  last_update:
    date:
      date: 20250618

The value is buried one level too deep and needs to be lifted up.

How it works
------------
-setValue with transform:copy reads from a dotted source path and writes to a
dotted destination path. Setting a dotted key to a scalar value replaces any
existing nested map at that key — so writing to last_update.date overwrites
the entire {date: 20250618} map in one step. No separate delete step needed.

Step 1 — Confirm the over-nesting:

  fmc -inspectProp last_update -dir ./docs

  Look for depth > expected and extra sub-keys like "date.date".

Step 2 — Lift the value up one level:

  Replace the specific paths with your actual property names.
  The pattern is always:  destination:transform:copy:source:always

  fmc -setValue "last_update.date:transform:copy:last_update.date.date:always" \
      -dir ./docs

  fmc shows a preview of every planned change before writing. Confirm you
  see the value moving from the deep path to the shallow path.

Step 3 — Verify the fix:

  fmc -inspectProp last_update -dir ./docs

  Depth should now be back to expected, and the extra sub-key should be gone.

Generalizing the pattern
------------------------
The same approach works for any over-nesting. Adapt the paths:

  Over-nested:  a.b.b: value   →   want: a.b: value
  Command:      -setValue "a.b:transform:copy:a.b.b:always"

  Over-nested:  meta.date.date.date: value   →   want: meta.date: value
  Command:      -setValue "meta.date:transform:copy:meta.date.date.date:always"

`)
}

func workflowAddFrontMatter() {
	fmt.Print(`Workflow: addFrontMatter — Find files missing front matter and add it
=====================================================================

Step 1 — Find files that have no front matter at all:

  fmc -analyze -dir ./docs

  Look for files reported as "no front matter". You can also grep directly:

    grep -rL "^---" ./docs

Step 2 — Add front matter to those files:

  At minimum, pass a template so fmc knows which keys to write:

    fmc -t template.json -createFrontMatter -dir ./docs

  Use -fmDefault to pre-fill useful fields instead of leaving them blank.
  Good candidates for auto-population:

    fmc -t template.json -createFrontMatter \
        -fmDefault "draft:true" \
        -dir ./docs

  Note: -fmDefault applies the same value to every file. For per-file computed
  values like unique UUIDs and today's date, run -genID and -setValue after:

    fmc -genID -dir ./docs
    fmc -setValue "last_update.date:computed:today:if_empty" -dir ./docs

Step 3 — Verify the result:

  fmc -analyze -dir ./docs

  All files should now show front matter present. Re-run with -issues-only
  to focus on anything still missing:

    fmc -t template.json -analyze -issues-only -dir ./docs

Bonus — Populate content fields with LLM suggestions:

  If you have an OpenAI API key configured in ~/.fmc/config.json, you can
  automatically generate title, description, tags, and keywords for the new
  files. Run the llmGenerate workflow for the full guide:

    fmc commonWorkflows llmGenerate

`)
}

func workflowLLMGenerate() {
	fmt.Print(`Workflow: llmGenerate — Generate and apply LLM-suggested front matter
======================================================================

How it works
------------
fmc uses OpenAI's Chat Completions API to read each file's markdown content
and suggest values for title, description, tags, and keywords. Results are
STAGED into a *_sources block in the front matter — nothing is written to
the real fields until you explicitly apply them. This lets you review before
committing to anything.

Staged data lives under keys like:
  title_sources.llm.gpt-4o.value
  description_sources.llm.gpt-4o.value
  tag_sources.llm.gpt-4o.tag_list
  keyword_sources.llm.gpt-4o.keyword_list
  tag_sources.llm.gpt-4o.date_last_generated   ← used for freshness checks

Prerequisites
-------------
Create ~/.fmc/config.json with your OpenAI API key:

  {
    "openai": {
      "api_key": "sk-...",
      "model": "gpt-4o"
    }
  }

Test that it works:
  fmc -llmTest

Step 1 — Stage LLM suggestions for all files:

  fmc -generateSources llm.gpt-4o -dir ./docs

  To generate only specific fields (cheaper/faster):
    fmc -generateSources llm.gpt-4o -llmFields tags,keywords -dir ./docs

  To skip files whose staged data is already fresh (e.g. generated within
  the last 7 days), but still regenerate if the page content is newer:
    fmc -generateSources llm.gpt-4o -llmSkipFresherThan 7 -llmRegenerateIfNewer -dir ./docs

Step 2 — Review the staged values:

  Open any file and inspect the *_sources block before applying anything.
  fmc -listEmptyForKey title -dir ./docs   # check which files still lack a title

Step 3 — Apply title and description:

  Only write if the field is currently empty (safest default):
    fmc -applyLLMGeneratedTitle "llm.gpt-4o:if_empty" -dir ./docs
    fmc -applyLLMGeneratedDescription "llm.gpt-4o:if_empty" -dir ./docs

  Always overwrite (useful when regenerating):
    fmc -applyLLMGeneratedTitle "llm.gpt-4o:always" -dir ./docs

Step 4 — Roll up tags and keywords:

  Merge LLM suggestions into the top-level tags field, preserving existing:
    fmc -rollup tags -rollupSources llm.gpt-4o -dir ./docs

  Same for keywords:
    fmc -rollup keywords -rollupSources llm.gpt-4o -dir ./docs

  Replace instead of merging (use -rollupNoPreserve):
    fmc -rollup tags,keywords -rollupSources llm.gpt-4o -rollupNoPreserve -dir ./docs

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
