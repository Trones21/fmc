# fmc — Front Matter Canonicalizer

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

#### Transform functions (`"source": "transform"`)

| fn | Output |
|---|---|
| `slug` | URL-safe slug derived from the source key (lowercase, spaces→dashes, special chars stripped) |

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

### Analysis

| Flag | Description |
|---|---|
| `-placementAudit` | Report where front matter appears in each file |
| `-listExtraProps` | Report keys present in files but absent from the template, with a summary count (useful for spotting typos and casing issues) |
| `-analyze` | Full analysis against the template *(not yet implemented)* |
| `-issues-only` | Only show files that have issues |
| `-verbose` | More detailed output |

### Fix options

Fix operations show a preview of all planned changes and prompt `[Y/n]` before writing anything.

| Flag | Description |
|---|---|
| `-allProps` | Add any missing template keys (uses policy for values) |
| `-fullConform` | Fully conform to the template: add missing, remove extras, fix order |
| `-fixOrder` | Reorder keys to match template order |
| `-removeExtraProps` | Remove keys not in the template |
| `-genID` | Generate an ID if the `id` key is missing or empty |

### Other

| Flag | Description |
|---|---|
| `-config <path>` | Path to a config JSON file |
| `-help` | Print flag reference |

---

## Example Usage

**Audit placement of front matter:**
```
fmc -dir ./docs --placementAudit
```

**Find extra/misspelled keys across a directory:**
```
fmc -t template.json -dir ./docs --listExtraProps
```

**Preview and apply fixes using a policy:**
```
fmc -t template.json -p policy.json -dir ./docs -allProps
```

**Fix a single file:**
```
fmc -t template.json -p policy.json -files ./docs/my-post.md -allProps
```
