# fmc — Front Matter Checker

A CLI tool for auditing and fixing YAML front matter in markdown files.

**IMPORTANT: fmc assumes files are under version control. It does not back up files before modifying them. Always review planned changes before confirming.**

## Build

```
go build
```

## Testing

**Unit tests** (colocated with source in `frontmatter/`):
```
go test ./...
```

**Integration tests** (run the compiled binary against `example-files/`):
```
go test -tags integration ./...
```

Integration tests are opt-in via the `integration` build tag so they don't slow down the default test run. They build the binary fresh each run and exercise the CLI flags end-to-end against the files in `example-files/`.

To run everything together:
```
go test -tags integration -v ./...
```

---

## Concepts

### Template

A JSON file that defines the expected front matter schema — which keys should exist. Values in the template are ignored; only the keys matter.

```json
{
  "id": "",
  "title": "",
  "last_update": "",
  "tags": ""
}
```

### Policy

A separate JSON file that defines *how* each key should be handled when fixing files. Properties not listed default to `preserve` (untouched).

```json
{
  "id":          { "action": "add_if_missing",     "source": "computed", "fn": "uuid" },
  "title":       { "action": "add_if_missing",     "source": "static" },
  "last_update": { "action": "overwrite_if_empty", "source": "computed", "fn": "today" },
  "slug":        { "action": "add_if_missing",     "source": "transform", "fn": "slug", "from": "title" },
  "Last_update": { "action": "rename_from",        "from": "Last_update" },
  "tags":        { "action": "preserve" }
}
```

#### Actions

| Action | Behaviour |
|---|---|
| `add_if_missing` | Add the key if it doesn't exist |
| `overwrite_always` | Always set the value |
| `overwrite_if_empty` | Set the value only if the key is missing or empty |
| `preserve` | Leave the key untouched (default) |
| `rename_from` | Rename an old key to this key, keeping the value; requires `"from"` |

#### Sources

| Source | Description |
|---|---|
| `static` | Use the literal `value` field |
| `computed` | Run a built-in function (see below) |
| `transform` | Derive a value from another property; requires `"from"` |

#### Computed functions (`"source": "computed"`)

| fn | Output |
|---|---|
| `today` | Current date as `YYYY-MM-DD` |
| `uuid` | Random UUID v4 |
| `path_segments` | Directory segments from the file path (drops root prefix and filename) |

#### Transform functions (`"source": "transform"`)

| fn | Output |
|---|---|
| `copy` | Copy the source value as-is |
| `slug` / `urlsafe` | URL-safe slug (lowercase, spaces→dashes, special chars stripped) |

---

## Configuration

User-level config lives at `~/.fmc/config.json`. This is where API keys and LLM preferences are stored — never in the project directory.

```json
{
  "openai": {
    "api_key": "sk-...",
    "model": "gpt-4o"
  },
  "llm": {
    "content_date_field": "last_update.date",
    "content_date_format": "YYYY-MM-DD"
  }
}
```

See `example-files/config.json` for a copy-paste starting point.

Supported OpenAI models: `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `gpt-3.5-turbo`

---

## Flags

### File selection (required)

| Flag | Description |
|---|---|
| `-dir <path>` | Directory to walk for `.md` files |
| `-files <a,b>` | Comma-separated list of specific files |

### Template & policy

| Flag | Description |
|---|---|
| `-t`, `-template <path>` | Path to the template JSON file |
| `-p`, `-policy <path>` | Path to the policy JSON file |

### Analysis (read-only)

| Flag | Description |
|---|---|
| `-placementAudit` | Report where front matter appears in each file |
| `-listExtraProps` | Report keys present in files but absent from the template |
| `-listMissingProps` | Report keys absent from files but present in the template |
| `-listEmpty` | Ranked table of which properties are most frequently empty |
| `-listEmptyDetails` | Per-file breakdown of empty properties (sortable with `-sortBy`) |
| `-listEmptyForKey <key>` | List files where a specific property is empty (repeatable) |
| `-sortBy <name\|count>` | Sort order for `-listEmptyDetails` |
| `-checkFormat <key:FORMAT>` | Report files where a property doesn't match a date/uuid format (repeatable) |
| `-checkType <key:type>` | Report files where a property is the wrong YAML type (repeatable) |
| `-listValues <key>` | List all distinct values for a property |
| `-listDateFormats <key>` | Summary of date formats seen in a property |
| `-listDateFormatsDetail <key>` | Per-file date format breakdown |
| `-analyze` | Full analysis against the template |
| `-analyzeOrder` | Check whether keys match template order |
| `-analyzeSEO` | Report SEO front matter coverage (requires `-plugin`) |
| `-plugin <docs\|blog>` | Docusaurus plugin for `-analyzeSEO` |
| `-inspectProp <key>` | Inspect nested YAML structure of a property (repeatable) |
| `-issues-only` | Only show files that have issues |
| `-verbose` | More detailed output |

### Single-property changes

Fix operations show a preview and prompt `[Y/n]` before writing anything.

| Flag | Description |
|---|---|
| `-setValue <key:source:value[:action]>` | Set a property value (static, computed, or transform) |
| `-replaceKey <OldKey:NewKey>` | Rename a key, preserving its value |
| `-createFrom <from:to[:action][:transform:fn]>` | Derive one key's value from another, with optional transform |
| `-genID` | Generate a UUID for the `id` key if missing or empty |
| `-genIDOverwriteInvalid` | Generate a UUID even if `id` exists but is not a valid UUID |
| `-tryCast <key:type>` | Cast a property to the specified YAML type |
| `-removeEmpty <key1,key2\|all>` | Remove empty-valued properties (CSV list or `all`) |

### Multi-property changes

| Flag | Description |
|---|---|
| `-createFrontMatter` | Add a front matter block to files that have none (requires `-t`) |
| `-fmDefault <key:value>` | Default value for a key during `-createFrontMatter` (repeatable) |
| `-onManualReview` | Limit `-createFrontMatter` to manual-review files only |
| `-keysToTop <key1,key2,...>` | Move listed keys to the top of front matter (CSV, ordered) |
| `-keysToBottom <key1,key2,...>` | Move listed keys to the bottom of front matter (CSV, ordered) |
| `-addMissingProps` | Add any template keys absent from a file (requires `-t`) |
| `-removeExtraProps` | Remove keys not in the template (requires `-t`) |
| `-allProps` | Add missing template keys using the policy for values |
| `-fullConform` | Add missing, remove extras, and fix order in one pass |
| `-fixOrder` | Reorder keys to match template order |

### Tags & keywords — source generation

Sources are staged values that feed into `tag_sources` / `keyword_sources` for review before being rolled up into the top-level `tags` / `keywords` fields.

| Flag | Description |
|---|---|
| `-generateSources <source>` | Populate `tag_sources` and `keyword_sources` from a source (`filepath` or `llm.<model>`) |
| `-rollup <tags\|keywords\|tags,keywords>` | Merge staged sources into the top-level field |
| `-rollupSources <source1,source2\|all>` | Which sources to include in rollup (dot notation for nested: `llm.gpt-4o`) |
| `-rollupNoPreserve` | Replace existing values instead of unioning with them |

### LLM generation (requires `~/.fmc/config.json`)

| Flag | Description |
|---|---|
| `-generateSources llm.<model>` | Send each file to OpenAI and stage results in `*_sources.llm.<model>` |
| `-llmFields <title,description,tags,keywords>` | CSV of fields to generate (default: all four) |
| `-llmSkipFresherThan <N>` | Skip files whose LLM source is less than N days old |
| `-llmRegenerateIfNewer` | Override skip when the content date field is newer than `date_last_generated` |
| `-applyLLMGeneratedTitle <source[:action]>` | Write staged title to top-level `title` key |
| `-applyLLMGeneratedDescription <source[:action]>` | Write staged description to top-level `description` key |
| `-llmTest` | Verify the OpenAI config and API key with a minimal test request |

Apply actions: *(none)* = add if missing, `if_empty` = write if absent or `""`, `always` = always overwrite.

### Display options

| Flag | Description |
|---|---|
| `-keepNonVariadicPathSegments <N>`, `-keepNVPS <N>` | Number of trailing path segments to show in output |

### Other

| Flag | Description |
|---|---|
| `-help` | Print flag reference |
| `-examples` | Print usage examples |

---

## Subcommands

```
fmc help <flag>          Detailed help for a specific flag
fmc commonWorkflows      Common multi-step cleanup sequences
fmc policy help          Policy file format reference
fmc policy list-functions  All built-in computed and transform functions
```

---

## Example Usage

**Audit placement of front matter:**
```
fmc -dir ./docs -placementAudit
```

**Ranked summary of empty properties:**
```
fmc -listEmpty -dir ./docs
```

**Per-file empty-property breakdown:**
```
fmc -listEmptyDetails -sortBy name -dir ./docs
```

**Find extra/misspelled keys across a directory:**
```
fmc -t template.json -dir ./docs -listExtraProps
```

**Check a date format and a UUID field:**
```
fmc -checkFormat "last_update.date:YYYY-MM-DD" -checkFormat "id:uuid" -dir ./docs
```

**Analyze SEO coverage (Docusaurus docs plugin):**
```
fmc -analyzeSEO -plugin docs -dir ./docs
```

**Generate IDs for files missing one:**
```
fmc -genID -dir ./docs
```

**Derive a slug from the title (only if slug is empty):**
```
fmc -createFrom title:slug:if_empty:transform:urlsafe -dir ./docs
```

**Move key order and add missing template keys:**
```
fmc -keysToTop id,title -keysToBottom tags,last_update -dir ./docs
fmc -t template.json -addMissingProps -dir ./docs
```

**Set a computed value with policy:**
```
fmc -setValue "last_update:computed:today:if_empty" -dir ./docs
```

**Remove empty keys:**
```
fmc -removeEmpty all -dir ./docs
```

**Generate LLM sources, then apply:**
```
# Stage LLM suggestions (skips files updated in the last 7 days unless content is newer):
fmc -generateSources llm.gpt-4o -llmSkipFresherThan 7 -llmRegenerateIfNewer -dir ./docs

# Apply staged title/description if missing:
fmc -applyLLMGeneratedTitle llm.gpt-4o:if_empty -dir ./docs
fmc -applyLLMGeneratedDescription llm.gpt-4o:if_empty -dir ./docs

# Roll up LLM tags into the tags field (union with existing):
fmc -rollup tags -rollupSources llm.gpt-4o -dir ./docs
```

**Test the OpenAI connection:**
```
fmc -llmTest
```

**Preview and apply fixes using a policy:**
```
fmc -t template.json -p policy.json -dir ./docs -allProps
```
