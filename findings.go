package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Finding is one machine-readable result from an analysis flag.
//
// Analysis output has always been human-readable tables, which are fine to read
// and miserable to consume — anything downstream ends up scraping text, and the
// scraper breaks when a column is widened. A Finding is the same information in
// a shape that pipes.
//
// Fields are deliberately flat. A consumer should be able to answer "what is
// wrong, where, and how badly" with jq and no schema knowledge.
type Finding struct {
	// File is the path as displayed, honouring -pathKeep.
	File string `json:"file"`
	// Check is the flag that produced this, e.g. "checkFormat", "listMissingProps".
	Check string `json:"check"`
	// Key is the front matter property involved, if the finding is about one.
	Key string `json:"key,omitempty"`
	// Severity is "error" for something unambiguously broken, "warn" for drift
	// worth reporting but not worth failing a build over.
	Severity string `json:"severity"`
	// Detail is a human sentence. Do not parse it; it is not stable.
	Detail string `json:"detail,omitempty"`
	// Value is the offending value, when there is one.
	Value string `json:"value,omitempty"`
	// Expected describes what would have been acceptable, when that is knowable.
	Expected string `json:"expected,omitempty"`
}

// JSON output modes.
const (
	JSONOff   = ""
	JSONLines = "ndjson"
	JSONArray = "array"
)

// ValidJSONModes lists the accepted -json values, for flag validation and help.
var ValidJSONModes = []string{JSONLines, JSONArray}

// FindingEmitter collects findings and writes them in the configured mode.
//
// The two modes differ in more than punctuation, which is why both exist:
//
//	ndjson  One JSON object per line, written as findings are produced. Constant
//	        memory regardless of how many files are scanned, output appears as
//	        work happens rather than at the end, and the stream stays valid if
//	        the run is interrupted — a killed scan still leaves every finding it
//	        had already emitted. Line-oriented, so grep, wc -l, head and
//	        `jq -c` all work directly. This is the mode for large runs, for
//	        piping into another process, and for anything you might want to
//	        watch live.
//
//	array   One JSON document containing every finding. Nothing is written until
//	        the run completes, and the whole set is held in memory first. In
//	        exchange the output is a single valid JSON value, so a consumer can
//	        do whole-set operations — group_by, sorting across all findings,
//	        length — in one jq expression without --slurp, and anything that
//	        expects to parse exactly one JSON value will accept it. This is the
//	        mode for a finite run whose output is read once, and for feeding
//	        tools that cannot handle a stream.
//
// When neither is set the emitter is disabled and callers print their usual
// human output.
type FindingEmitter struct {
	mode string
	w    io.Writer
	enc  *json.Encoder
	// buf holds findings in array mode. Unused for ndjson, which never buffers.
	buf []Finding
	// count tracks emitted findings so callers can report an exit status without
	// re-walking the buffer, which does not exist in ndjson mode.
	count  int
	errors int
}

// NewFindingEmitter validates mode and returns an emitter. An empty mode yields
// a disabled emitter, which is the normal path.
func NewFindingEmitter(mode string, w io.Writer) (*FindingEmitter, error) {
	switch mode {
	case JSONOff:
		return &FindingEmitter{mode: JSONOff}, nil
	case JSONLines, JSONArray:
	default:
		return nil, fmt.Errorf("invalid -json %q: expected one of %s",
			mode, strings.Join(ValidJSONModes, ", "))
	}
	if w == nil {
		w = os.Stdout
	}
	e := &FindingEmitter{mode: mode, w: w}
	if mode == JSONLines {
		e.enc = json.NewEncoder(w)
	}
	return e, nil
}

// Enabled reports whether JSON output is on. Callers use this to suppress their
// human-readable printing.
func (e *FindingEmitter) Enabled() bool { return e != nil && e.mode != JSONOff }

// Add records a finding. In ndjson mode it is written immediately; in array mode
// it is buffered until Flush.
func (e *FindingEmitter) Add(f Finding) {
	if !e.Enabled() {
		return
	}
	if f.Severity == "" {
		f.Severity = "error"
	}
	e.count++
	if f.Severity == "error" {
		e.errors++
	}
	if e.mode == JSONLines {
		// An encode failure here means stdout is gone (closed pipe, full disk).
		// Nothing useful can be reported through the same broken stream, and the
		// run should not be aborted mid-scan, so this is deliberately dropped.
		_ = e.enc.Encode(f)
		return
	}
	e.buf = append(e.buf, f)
}

// Count returns how many findings were emitted.
func (e *FindingEmitter) Count() int {
	if e == nil {
		return 0
	}
	return e.count
}

// ErrorCount returns how many emitted findings had severity "error".
func (e *FindingEmitter) ErrorCount() int {
	if e == nil {
		return 0
	}
	return e.errors
}

// Flush writes buffered output. It is a no-op for ndjson, which has already
// written everything, and must be called exactly once for array mode.
//
// An empty run in array mode emits "[]" rather than nothing, so a consumer can
// always parse the output rather than special-casing silence.
func (e *FindingEmitter) Flush() error {
	if !e.Enabled() || e.mode != JSONArray {
		return nil
	}
	if e.buf == nil {
		e.buf = []Finding{}
	}
	enc := json.NewEncoder(e.w)
	enc.SetIndent("", "  ")
	return enc.Encode(e.buf)
}
