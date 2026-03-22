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
	printFlag(out, "issues-only", "")
	printFlag(out, "verbose", "")
	printFlag(out, "analyze", "")
	printFlag(out, "analyzeOrder", "")
	printFlag(out, "inspectProp", "<key>")

	section(out, "Make Changes — Single Property:")
	printFlag(out, "setValue", "<key:source:value[:action]>")
	printFlag(out, "replaceKey", "<OldKey:NewKey>")
	printFlag(out, "createSlug", "<FromKey:ToKey[:action]>")
	printFlag(out, "genID", "")
	printFlag(out, "removeEmpty", "<propertyName>")

	section(out, "Make Changes — Multi Property:")
	printFlag(out, "createFrontMatter", "")
	printFlag(out, "fmDefault", "<key:value>")
	printFlag(out, "addMissingProps", "")
	printFlag(out, "removeExtraProps", "")
	printFlag(out, "allProps", "")
	printFlag(out, "fullConform", "")
	printFlag(out, "fixOrder", "")

	section(out, "Display Options:")
	printFlag(out, "keepNonVariadicPathSegments", "<N>")
	printFlag(out, "keepNVPS", "<N>")

	section(out, "Other:")
	printFlag(out, "help", "")

	fmt.Fprintln(out, `
Examples:
  Audit front matter placement:
    fmc -dir ./docs -placementAudit

  Find extra/misspelled keys across a directory:
    fmc -t template.json -dir ./docs -listExtraProps

  Add missing template keys (empty value):
    fmc -t template.json -addMissingProps -dir ./docs

  Remove keys not in the template:
    fmc -t template.json -removeExtraProps -dir ./docs

  Set a value (static, computed, or llm):
    fmc -setValue "last_update:computed:today:if_empty" -dir ./docs

  Policy subcommand help:
    fmc policy help
    fmc policy list-functions

  Flag-specific help:
    fmc help setValue
    fmc help addMissingProps
    fmc help removeExtraProps
    fmc help createSlug
    fmc help replaceKey`)
}

func runHelpTopic(topic string) {
	switch topic {
	case "createSlug":
		fmt.Print(`-createSlug FromKey:ToKey[:action]

  Derives a URL-safe slug from an existing front matter property and writes it
  to a new (or existing) property. Action controls when the value is written.

Actions:
  (none)     add_if_missing — only set if the destination key is absent (default)
  if_empty   overwrite_if_empty — set if the destination is absent or ""
  always     overwrite_always — always overwrite the destination

Examples:
  Add a slug from title, only if slug is missing:
    fmc -createSlug title:slug -dir ./docs

  Overwrite slug whenever it is missing or empty:
    fmc -createSlug title:slug:if_empty -dir ./docs

  Always regenerate the slug from title:
    fmc -createSlug title:slug:always -dir ./docs

  Multiple slugs in one pass:
    fmc -createSlug title:slug -createSlug name:id_slug -dir ./docs

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
  static     Use the literal string as the value
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
	default:
		fmt.Printf("no help topic %q\n\n", topic)
		fmt.Println("Only a subset of flags currently have dedicated help. Available topics:")
		fmt.Println("  fmc help setValue")
		fmt.Println("  fmc help addMissingProps")
		fmt.Println("  fmc help removeExtraProps")
		fmt.Println("  fmc help createSlug")
		fmt.Println("  fmc help replaceKey")
		fmt.Println("  fmc help createFrontMatter")
		fmt.Println("  fmc help inspectProp")
		fmt.Println()
		fmt.Println("For the full flag list run: fmc help")
		os.Exit(1)
	}
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
