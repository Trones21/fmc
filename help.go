package main

import (
	"flag"
	"fmt"
	"os"
)

func printHelp() {
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, "Usage: fmc [flags]")
	flag.PrintDefaults()
	fmt.Fprintln(out, `
Examples:
  Audit front matter placement:
    fmc -dir ./docs -placementAudit

  Find extra/misspelled keys across a directory:
    fmc -t template.json -dir ./docs -listExtraProps

  Preview and apply fixes using a policy:
    fmc -t template.json -p policy.json -dir ./docs -allProps

  Fix a single file:
    fmc -t template.json -p policy.json -files ./docs/my-post.md -allProps

  Policy subcommand help:
    fmc policy help
    fmc policy list-functions

  Flag-specific help:
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
	default:
		fmt.Printf("no help topic %q\n\n", topic)
		fmt.Println("Only a subset of flags currently have dedicated help. Available topics:")
		fmt.Println("  fmc help createSlug")
		fmt.Println("  fmc help replaceKey")
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
		fmt.Println("Transform functions  (\"source\": \"transform\")  — require \"from\": \"<key>\"")
		fmt.Println()
		fmt.Println("  slug             URL-safe slug (lowercase, spaces→dashes, special chars stripped)")
	default:
		fmt.Printf("unknown policy command %q\n", args[0])
		fmt.Println("Run 'fmc policy' to see available commands.")
		os.Exit(1)
	}
}
