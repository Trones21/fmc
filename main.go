package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Trones21/fmc/frontmatter"
)

// repeatableFlag allows a flag to be specified multiple times, collecting all values.
type repeatableFlag []string

func (f *repeatableFlag) String() string { return strings.Join(*f, ", ") }
func (f *repeatableFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

type Config struct {
	ValueInsertion map[string]any `json:"valueInsertion"`
}

type FrontMatterChecker struct {
	TemplateFile       string
	Dir                string
	Files              []string
	ConfigFile         string
	PolicyFile         string
	FixOptions         map[string]bool
	AnalyzeOnly        bool
	PlacementAuditOnly bool
	GenID                    bool
	GenIDOverwriteInvalid    bool
	Config             Config

	IssuesOnly       bool
	Verbose          bool
	ListExtraProps   bool
	ListMissingProps bool
	ReplaceKeys      repeatableFlag // each entry: "OldKey:NewKey"
	CreateFrom       repeatableFlag // each entry: "FromKey:ToKey[:action][:transform:fn]"
	SetValues        repeatableFlag // each entry: "key:source:value[:action]"
	AddMissingProps  bool
	RemoveExtraProps bool
	RemoveEmpty          string // CSV of property names, or "all"
	ListEmpty            bool           // scan all keys, show empty-counts table
	ListEmptyDetails     bool           // per-file breakdown: file | # empty | keys
	ListEmptyForKey      repeatableFlag // each entry: property name
	SortBy               string         // "name" or "count" for list-empty outputs
	InspectProps         repeatableFlag // each entry: property name
	PathKeep             int            // -1 = full path, 0 = filename only, N = last N dirs + filename
	CreateFrontMatter    bool
	OnManualReview       bool
	FmDefaults           repeatableFlag // each entry: "key:value"
	AnalyzeOrder         bool
	AnalyzeSEO           bool
	Plugin               string         // "docs" or "blog"
	CheckFormats         repeatableFlag // each entry: "key:FORMAT"
	CheckTypes           repeatableFlag // each entry: "key:type"
	TryCast              repeatableFlag // each entry: "key:type"
	KeysToTop            string // CSV of keys to move to the front, in order
	KeysToBottom         string // CSV of keys to move to the end, in order
	ListValues           repeatableFlag // each entry: property name
	ListDateFormats       repeatableFlag // each entry: property name
	ListDateFormatsDetail repeatableFlag // each entry: property name
	GenerateSources       string         // source name to generate: "filepath"
	Rollup                string         // CSV: "tags", "keywords", or "tags,keywords"
	RollupSources         string         // CSV of source paths, or "all"
	RollupNoPreserve      bool           // replace existing tags/keywords instead of unioning
}

func main() {
	// policy subcommand intercepted before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "policy" {
		runPolicyCommand(os.Args[2:])
		return
	}

	// commonWorkflows subcommand
	if len(os.Args) > 1 && os.Args[1] == "commonWorkflows" {
		printCommonWorkflows()
		return
	}

	flag.Usage = printHelp

	checker := &FrontMatterChecker{
		FixOptions: make(map[string]bool),
	}

	// Register flags
	////// Front Matter Template /////
	flag.StringVar(&checker.TemplateFile, "template", "", "Path to the front matter template file")
	flag.StringVar(&checker.TemplateFile, "t", "", "Alias for -template")

	////// Policy/Config - Vestigial - May delete later /////
	// flag.StringVar(&checker.ConfigFile, "config", "", "Path to the configuration JSON file")
	// flag.StringVar(&checker.PolicyFile, "policy", "", "Path to the property policy JSON file")
	// flag.StringVar(&checker.PolicyFile, "p", "", "Alias for -policy")

	////// Dir/files to operate on /////
	flag.StringVar(&checker.Dir, "dir", "", "Directory containing markdown files")
	files := flag.String("files", "", "Comma-separated list of files to analyze/fix")

	////// List/analyze - Do not rewrite front matter /////
	issuesOnly := flag.Bool("issues-only", false, "Show only files with issues")
	verbose := flag.Bool("verbose", false, "Show more detailed analysis output")
	placementAudit := flag.Bool("placementAudit", false, "Audit front matter placement only")
	analyzeOnly := flag.Bool("analyze", false, "Analyze the files without making changes")
	listExtraProps := flag.Bool("listExtraProps", false, "List properties not defined in the template")
	listMissingProps := flag.Bool("listMissingProps", false, "List template properties missing from each file")
	analyzeOrder := flag.Bool("analyzeOrder", false, "Check whether each file's front matter keys match the template order (requires -t)")
	analyzeSEO := flag.Bool("analyzeSEO", false, "Analyze SEO-relevant front matter properties (requires -plugin)")
	plugin := flag.String("plugin", "", "Docusaurus plugin to target for SEO analysis: docs or blog")
	var checkFormats repeatableFlag
	flag.Var(&checkFormats, "checkFormat", "Check that a property matches a date format, e.g. last_update.date:YYYY-MM-DD (repeatable)")
	var checkTypes repeatableFlag
	flag.Var(&checkTypes, "checkType", "List files where a property exists but is the wrong type, e.g. disable:bool (repeatable)")
	var listValues repeatableFlag
	flag.Var(&listValues, "listValues", "List all unique values and their counts for a property (repeatable)")
	var listDateFormats repeatableFlag
	flag.Var(&listDateFormats, "listDateFormats", "List which date formats are in use for a property, with counts (repeatable)")
	var listDateFormatsDetail repeatableFlag
	flag.Var(&listDateFormatsDetail, "listDateFormatsDetail", "Per-file table: file | format | length | value (greppable; repeatable)")
	var tryCast repeatableFlag
	flag.Var(&tryCast, "tryCast", "Cast a property's value to the target type, e.g. disable:bool (repeatable)")
	keysToTop := flag.String("keysToTop", "", "CSV of keys to move to the front of the front matter, in order (e.g. id,title,slug)")
	keysToBottom := flag.String("keysToBottom", "", "CSV of keys to move to the end of the front matter, in order (e.g. tags,last_update)")
	generateSources := flag.String("generateSources", "", "Populate tag_sources and keyword_sources from a source: filepath")
	rollup := flag.String("rollup", "", "Roll up staged sources into tags/keywords: tags, keywords, or tags,keywords")
	rollupSources := flag.String("rollupSources", "", "CSV of sources to roll up, or 'all' (e.g. filepath,llm.gpt-4o)")
	rollupNoPreserve := flag.Bool("rollupNoPreserve", false, "Replace existing tags/keywords instead of unioning with them")

	///// Make Changes to Front Matter ///////
	//Single Property CRUD
	genID := flag.Bool("genID", false, "Generate a UUID for the id property when it is missing or empty")
	genIDOverwriteInvalid := flag.Bool("genIDOverwriteInvalid", false, "Also overwrite id values that are not valid UUIDs (use with -genID)")
	flag.Var(&checker.ReplaceKeys, "replaceKey", "Rename a key, keeping its value (repeatable; see: fmc help replaceKey)")
	flag.Var(&checker.CreateFrom, "createFrom", "Derive a key from another key's value, with optional transform (repeatable; see: fmc help createFrom)")
	flag.Var(&checker.SetValues, "setValue", "Set a property via static, computed, or llm source (repeatable; see: fmc help setValue)")
	flag.StringVar(&checker.RemoveEmpty, "removeEmpty", "", "Remove properties with empty values: 'all' or comma-separated key list (e.g. title,description)")
	listEmpty := flag.Bool("listEmpty", false, "Show counts of empty properties across all keys in all files")
	listEmptyDetails := flag.Bool("listEmptyDetails", false, "Per-file breakdown: file | # empty | empty keys (sortable with -sortBy)")
	flag.Var(&checker.ListEmptyForKey, "listEmptyForKey", "List files where a specific property is empty or whitespace (repeatable)")
	sortBy := flag.String("sortBy", "count", "Sort order for -listEmptyDetails: 'name' or 'count' (default: count)")
	flag.Var(&checker.InspectProps, "inspectProp", "Inspect nested YAML structure of a property across files (repeatable)")

	///// Display Options /////
	keep := flag.Int("keepNonVariadicPathSegments", -1, "Trailing path segments to show in output (-1 = full, 0 = filename only, N = N dirs + filename)")
	flag.IntVar(keep, "keepNVPS", -1, "Alias for -keepNonVariadicPathSegments")

	//Multi Property CRUD
	fixFullConform := flag.Bool("fullConform", false, "Fully conform the front matter to the template")
	fixAllProps := flag.Bool("allProps", false, "Ensure all properties in the template exist in the front matter")
	addMissingProps := flag.Bool("addMissingProps", false, "Add any template keys missing from each file (empty value)")
	removeExtraProps := flag.Bool("removeExtraProps", false, "Remove properties not defined in the template")
	createFrontMatter := flag.Bool("createFrontMatter", false, "Add front matter to files that are missing it (requires -t)")
	onManualReview := flag.Bool("onManualReview", false, "Used with -createFrontMatter: operate only on files flagged as manual_review")
	flag.Var(&checker.FmDefaults, "fmDefault", "Default value for a property during -createFrontMatter (repeatable; key:value)")

	//Other
	fixOrder := flag.Bool("fixOrder", false, "Reorder properties to match the template")

	///// Help/Examples /////
	help := flag.Bool("help", false, "Display help information")
	examples := flag.Bool("examples", false, "Show usage examples")

	// help subcommand intercepted after flag registration so PrintDefaults works
	if len(os.Args) > 1 && os.Args[1] == "help" {
		if len(os.Args) > 2 {
			runHelpTopic(os.Args[2])
		} else {
			printHelp()
		}
		return
	}

	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if *examples {
		printExamples()
		return
	}

	checker.PathKeep = *keep

	// Audit/Analysis Modes
	checker.PlacementAuditOnly = *placementAudit
	checker.AnalyzeOnly = *analyzeOnly

	// Analysis output
	checker.IssuesOnly = *issuesOnly
	checker.Verbose = *verbose

	// Modification
	checker.FixOptions["fullConform"] = *fixFullConform
	checker.FixOptions["allProps"] = *fixAllProps
	checker.FixOptions["fixOrder"] = *fixOrder
	checker.GenID = *genID
	checker.GenIDOverwriteInvalid = *genIDOverwriteInvalid
	checker.CreateFrontMatter = *createFrontMatter
	checker.OnManualReview = *onManualReview
	checker.AnalyzeOrder = *analyzeOrder
	checker.AnalyzeSEO = *analyzeSEO
	checker.Plugin = *plugin
	checker.CheckFormats = checkFormats
	checker.CheckTypes = checkTypes
	checker.ListValues = listValues
	checker.ListDateFormats = listDateFormats
	checker.ListDateFormatsDetail = listDateFormatsDetail
	checker.TryCast = tryCast
	checker.KeysToTop = *keysToTop
	checker.KeysToBottom = *keysToBottom
	checker.ListExtraProps = *listExtraProps
	checker.ListMissingProps = *listMissingProps
	checker.AddMissingProps = *addMissingProps
	checker.RemoveExtraProps = *removeExtraProps
	checker.ListEmpty = *listEmpty
	checker.ListEmptyDetails = *listEmptyDetails
	checker.SortBy = *sortBy
	checker.GenerateSources = *generateSources
	checker.Rollup = *rollup
	checker.RollupSources = *rollupSources
	checker.RollupNoPreserve = *rollupNoPreserve

	if *files != "" {
		checker.Files = strings.Split(*files, ",")
	}

	if err := checker.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func (fmc *FrontMatterChecker) Run() error {
	if fmc.ConfigFile != "" {
		if err := fmc.loadConfig(); err != nil {
			return err
		}
	}

	filesToProcess, err := fmc.getFiles()
	if err != nil {
		return err
	}

	if fmc.PlacementAuditOnly {
		return fmc.auditPlacement(filesToProcess)
	}

	if len(fmc.InspectProps) > 0 {
		return fmc.inspectProps(filesToProcess)
	}

	if fmc.GenID {
		return fmc.runGenID(filesToProcess)
	}

	if len(fmc.ReplaceKeys) > 0 {
		return fmc.replaceKeys(filesToProcess)
	}

	if len(fmc.CreateFrom) > 0 {
		return fmc.createFrom(filesToProcess)
	}

	if fmc.GenerateSources != "" {
		return fmc.runGenerateSources(filesToProcess)
	}

	if fmc.Rollup != "" {
		return fmc.runRollup(filesToProcess)
	}

	if len(fmc.SetValues) > 0 {
		return fmc.setValues(filesToProcess)
	}

	if fmc.RemoveEmpty != "" {
		return fmc.removeEmpty(filesToProcess)
	}

	if fmc.ListEmpty {
		return fmc.listEmptyAll(filesToProcess)
	}

	if fmc.ListEmptyDetails {
		return fmc.listEmptyDetails(filesToProcess)
	}

	if len(fmc.ListEmptyForKey) > 0 {
		return fmc.listEmptyForKey(filesToProcess)
	}

	if fmc.AnalyzeSEO {
		if fmc.Plugin == "" {
			return fmt.Errorf("-analyzeSEO requires -plugin (docs or blog)")
		}
		return fmc.analyzeSEO(filesToProcess)
	}

	if len(fmc.CheckFormats) > 0 {
		return fmc.runCheckFormats(filesToProcess)
	}

	if len(fmc.CheckTypes) > 0 {
		return fmc.runCheckTypes(filesToProcess)
	}

	if len(fmc.ListValues) > 0 {
		return fmc.runListValues(filesToProcess)
	}

	if len(fmc.ListDateFormats) > 0 {
		return fmc.runListDateFormats(filesToProcess)
	}

	if len(fmc.ListDateFormatsDetail) > 0 {
		return fmc.runListDateFormatsDetail(filesToProcess)
	}

	if len(fmc.TryCast) > 0 {
		return fmc.runTryCast(filesToProcess)
	}

	if len(fmc.KeysToTop) > 0 || len(fmc.KeysToBottom) > 0 {
		return fmc.runReorder(filesToProcess)
	}

	template, err := fmc.loadTemplate()
	if err != nil {
		return err
	}

	var templateKeys []string
	if (fmc.AnalyzeOrder || fmc.FixOptions["fixOrder"] || fmc.AnalyzeOnly) && fmc.TemplateFile != "" {
		templateKeys, err = fmc.loadTemplateKeyOrder()
		if err != nil {
			return err
		}
	}

	var policies []frontmatter.PropertyPolicy
	if fmc.PolicyFile != "" {
		policies, err = frontmatter.LoadPolicy(fmc.PolicyFile)
		if err != nil {
			return err
		}
	}

	if fmc.ListExtraProps {
		return fmc.listExtraProps(filesToProcess, template)
	}

	if fmc.ListMissingProps {
		return fmc.listMissingProps(filesToProcess, template)
	}

	if fmc.AddMissingProps {
		return fmc.addMissingProps(filesToProcess, template)
	}

	if fmc.RemoveExtraProps {
		return fmc.removeExtraProps(filesToProcess, template)
	}

	if fmc.CreateFrontMatter {
		return fmc.createFrontMatter(filesToProcess, template)
	}

	if fmc.AnalyzeOrder {
		return fmc.analyzeOrder(filesToProcess, template, templateKeys)
	}

	if fmc.AnalyzeOnly {
		return fmc.analyzeFiles(filesToProcess, template, templateKeys)
	}

	return fmc.fixFiles(filesToProcess, template, policies)
}

func (fmc *FrontMatterChecker) loadTemplate() (map[string]any, error) {
	file, err := os.Open(fmc.TemplateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open template file: %v", err)
	}
	defer file.Close()

	template := make(map[string]any)
	if err := json.NewDecoder(file).Decode(&template); err != nil {
		return nil, fmt.Errorf("failed to parse template file: %v", err)
	}

	return template, nil
}

func (fmc *FrontMatterChecker) loadTemplateKeyOrder() ([]string, error) {
	data, err := os.ReadFile(fmc.TemplateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("template must be a JSON object")
	}
	var keys []string
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read template key: %w", err)
		}
		keys = append(keys, key.(string))
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, fmt.Errorf("failed to read template value: %w", err)
		}
	}
	return keys, nil
}

func (fmc *FrontMatterChecker) analyzeOrder(files []string, template map[string]any, templateKeys []string) error {
	type fileResult struct {
		path     string
		status   string // "ok", "out_of_order", "excluded"
	}

	var results []fileResult
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}

		missing, err := frontmatter.FindMissingProps(string(content), template)
		if err != nil || len(missing) > 0 {
			results = append(results, fileResult{path: file, status: "excluded"})
			continue
		}

		fileKeys, err := frontmatter.GetFrontMatterKeyOrder(string(content))
		if err != nil {
			fmt.Printf("warning: %s: %v\n", file, err)
			continue
		}

		if frontmatter.IsOrderedByTemplate(fileKeys, templateKeys) {
			results = append(results, fileResult{path: file, status: "ok"})
		} else {
			results = append(results, fileResult{path: file, status: "out_of_order"})
		}
	}

	tbl := NewTable("File", "Order")
	inOrder, outOfOrder, excluded := 0, 0, 0
	for _, r := range results {
		switch r.status {
		case "excluded":
			excluded++
			if !fmc.IssuesOnly {
				tbl.AddRow(displayPath(r.path, fmc.PathKeep), "excluded")
			}
		case "ok":
			inOrder++
			if !fmc.IssuesOnly {
				tbl.AddRow(displayPath(r.path, fmc.PathKeep), "ok")
			}
		case "out_of_order":
			outOfOrder++
			tbl.AddRow(displayPath(r.path, fmc.PathKeep), "out_of_order")
		}
	}
	tbl.Print()

	fmt.Printf("\nSummary: %d in order, %d out of order, %d excluded (missing template properties)\n",
		inOrder, outOfOrder, excluded)
	return nil
}

var seoKeysByPlugin = map[string][]string{
	"docs": {"title", "description", "keywords", "image", "slug"},
	"blog": {"title", "title_meta", "description", "keywords", "image", "slug"},
}

func (fmc *FrontMatterChecker) analyzeSEO(files []string) error {
	keys, ok := seoKeysByPlugin[fmc.Plugin]
	if !ok {
		return fmt.Errorf("unknown plugin %q: expected docs or blog", fmc.Plugin)
	}

	type counts struct{ missing, empty int }
	tally := make(map[string]*counts, len(keys))
	for _, k := range keys {
		tally[k] = &counts{}
	}

	total := len(files)
	excluded := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			total--
			continue
		}

		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil {
			fmt.Printf("warning: %s: %v\n", file, err)
			total--
			continue
		}

		if isBoolTrue(fm["unlisted"]) || isBoolTrue(fm["draft"]) {
			excluded++
			continue
		}

		for _, k := range keys {
			val, exists := fm[k]
			if !exists {
				tally[k].missing++
			} else if isEmptyVal(val) {
				tally[k].empty++
			}
		}
	}

	analyzed := total - excluded
	fmt.Printf("Total Files: %d\n", total)
	fmt.Printf("Unlisted or Draft Files: %d\n", excluded)
	fmt.Printf("SEO Analyzed Files: %d\n", analyzed)
	fmt.Println()
	fmt.Printf("SEO Analysis — plugin: %s\n\n", fmc.Plugin)
	tbl := NewTable("SEO Property", "Missing", "Empty")
	for _, k := range keys {
		c := tally[k]
		tbl.AddRow(k, fmt.Sprintf("%d", c.missing), fmt.Sprintf("%d", c.empty))
	}
	tbl.Print()
	return nil
}

// parseStaticValue coerces a CLI string to bool or int when unambiguous,
// falling back to the raw string otherwise.
func parseStaticValue(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return s
}

// userFormatToGoLayout converts a user-friendly date format string to a Go
// time layout. Supported tokens: YYYY MM DD HH mm ss
func userFormatToGoLayout(format string) string {
	r := format
	r = strings.ReplaceAll(r, "YYYY", "2006")
	r = strings.ReplaceAll(r, "MM", "01")
	r = strings.ReplaceAll(r, "DD", "02")
	r = strings.ReplaceAll(r, "HH", "15")
	r = strings.ReplaceAll(r, "mm", "04")
	r = strings.ReplaceAll(r, "ss", "05")
	return r
}

var reUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validateFormat(format, layout, s string) bool {
	if format == "uuid" {
		return reUUID.MatchString(s)
	}
	_, err := time.Parse(layout, s)
	return err == nil
}

func yamlTypeName(v any) string {
	switch v.(type) {
	case bool:
		return "bool"
	case int:
		return "int"
	case float64:
		return "float"
	case string:
		return "string"
	case []any:
		return "list"
	case map[string]any:
		return "map"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func matchesType(v any, typeName string) bool {
	switch typeName {
	case "bool":
		_, ok := v.(bool)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "int":
		_, ok := v.(int)
		return ok
	case "float":
		_, ok := v.(float64)
		return ok
	case "list", "array":
		_, ok := v.([]any)
		return ok
	case "map", "object":
		_, ok := v.(map[string]any)
		return ok
	}
	return false
}

func castValue(val any, targetType string) (any, error) {
	if matchesType(val, targetType) {
		return val, nil // already correct
	}
	s, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("cannot cast %s to %s (only string source is supported)", yamlTypeName(val), targetType)
	}
	switch targetType {
	case "bool":
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		return nil, fmt.Errorf("cannot cast string %q to bool (expected \"true\" or \"false\")", s)
	case "int":
		i, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("cannot cast string %q to int", s)
		}
		return i, nil
	case "float":
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return nil, fmt.Errorf("cannot cast string %q to float", s)
		}
		return f, nil
	case "string":
		return fmt.Sprintf("%v", val), nil
	}
	return nil, fmt.Errorf("unsupported target type %q", targetType)
}

func (fmc *FrontMatterChecker) runTryCast(files []string) error {
	type castSpec struct {
		key      string
		typeName string
	}
	type fileCast struct {
		file    string
		key     string
		oldVal  any
		newVal  any
		oldType string
	}

	specs := make([]castSpec, 0, len(fmc.TryCast))
	for _, entry := range fmc.TryCast {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -tryCast %q: expected key:type (e.g. disable:bool)", entry)
		}
		specs = append(specs, castSpec{key: parts[0], typeName: parts[1]})
	}

	var pending []fileCast
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil || len(fm) == 0 {
			continue
		}
		for _, spec := range specs {
			val, ok := frontmatter.NestedGet(fm, frontmatter.KeyPath(spec.key))
			if !ok {
				continue // absent — skip
			}
			if matchesType(val, spec.typeName) {
				continue // already correct type
			}
			newVal, err := castValue(val, spec.typeName)
			if err != nil {
				fmt.Printf("  warning: %s — %s: %v\n", displayPath(file, fmc.PathKeep), spec.key, err)
				continue
			}
			pending = append(pending, fileCast{
				file:    file,
				key:     spec.key,
				oldVal:  val,
				newVal:  newVal,
				oldType: yamlTypeName(val),
			})
		}
	}

	if len(pending) == 0 {
		fmt.Println("No values need casting.")
		return nil
	}

	fmt.Printf("Will cast %d value(s):\n\n", len(pending))
	for _, c := range pending {
		fmt.Printf("  %s  %s: %v (%s) → %v\n",
			displayPath(c.file, fmc.PathKeep), c.key, c.oldVal, c.oldType, c.newVal)
	}

	fmt.Print("\nApply these changes? [Y/n]: ")
	var response string
	fmt.Scanln(&response)
	if response != "" && strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	// Group by file to apply all casts in a single write per file
	byFile := make(map[string][]fileCast)
	for _, c := range pending {
		byFile[c.file] = append(byFile[c.file], c)
	}
	for file, casts := range byFile {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("error: %s: %v\n", file, err)
			continue
		}
		policies := make([]frontmatter.PropertyPolicy, 0, len(casts))
		template := make(map[string]any, len(casts))
		for _, c := range casts {
			template[c.key] = ""
			policies = append(policies, frontmatter.PropertyPolicy{
				Key:         c.key,
				Action:      frontmatter.ActionOverwriteAlways,
				Source:      frontmatter.SourceStatic,
				StaticValue: c.newVal,
			})
		}
		plan, err := frontmatter.PlanChanges(file, string(content), template, policies)
		if err != nil {
			fmt.Printf("error: %s: %v\n", file, err)
			continue
		}
		if err := frontmatter.ApplyChangePlan(plan); err != nil {
			fmt.Printf("error: %s: %v\n", file, err)
		} else {
			fmt.Printf("  wrote %s\n", displayPath(file, fmc.PathKeep))
		}
	}
	return nil
}

var knownDateFormats = []struct{ name, layout string }{
	{"YYYYMMDD", "20060102"},
	{"YYYY-MM-DD", "2006-01-02"},
	{"YYYY/MM/DD", "2006/01/02"},
	{"DD-MM-YYYY", "02-01-2006"},
	{"DD/MM/YYYY", "02/01/2006"},
	{"MM/DD/YYYY", "01/02/2006"},
	{"RFC3339", time.RFC3339},
}

func detectDateFormat(s string) string {
	for _, f := range knownDateFormats {
		if _, err := time.Parse(f.layout, s); err == nil {
			return f.name
		}
	}
	return ""
}

func (fmc *FrontMatterChecker) runListDateFormats(files []string) error {
	for _, key := range fmc.ListDateFormats {
		counts := make(map[string]int)
		empty := 0
		unparseableByLen := make(map[int]int)

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("warning: could not read %s: %v\n", file, err)
				continue
			}
			fm, err := frontmatter.GetFrontMatterMap(string(content))
			if err != nil || len(fm) == 0 {
				continue
			}
			val, ok := frontmatter.NestedGet(fm, frontmatter.KeyPath(key))
			if !ok {
				continue
			}
			s, isStr := val.(string)
			if !isStr {
				s = fmt.Sprintf("%v", val)
			}
			if strings.TrimSpace(s) == "" {
				empty++
				continue
			}
			if format := detectDateFormat(s); format != "" {
				counts[format]++
			} else {
				unparseableByLen[len(s)]++
			}
		}

		unparseable := 0
		for _, c := range unparseableByLen {
			unparseable += c
		}

		type entry struct {
			name  string
			count int
		}
		entries := make([]entry, 0, len(counts))
		for name, c := range counts {
			entries = append(entries, entry{name, c})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].count != entries[j].count {
				return entries[i].count > entries[j].count
			}
			return entries[i].name < entries[j].name
		})

		fmt.Printf("Date formats for %q:\n", key)
		for _, e := range entries {
			fmt.Printf("  %-20s %d\n", e.name, e.count)
		}
		if empty > 0 {
			fmt.Printf("  %-20s %d\n", "(empty)", empty)
		}
		if unparseable > 0 {
			fmt.Printf("  %-20s %d\n", "(unrecognized)", unparseable)

			// secondary breakdown by value length
			type lenEntry struct{ length, count int }
			lenEntries := make([]lenEntry, 0, len(unparseableByLen))
			for l, c := range unparseableByLen {
				lenEntries = append(lenEntries, lenEntry{l, c})
			}
			sort.Slice(lenEntries, func(i, j int) bool {
				return lenEntries[i].length < lenEntries[j].length
			})
			fmt.Println("\n  Unrecognized values by length:")
			for _, e := range lenEntries {
				fmt.Printf("    %d chars%s%d\n", e.length, strings.Repeat(" ", max(1, 12-len(fmt.Sprintf("%d chars", e.length)))), e.count)
			}
		}
		fmt.Printf("  Tip: fmc -listDateFormatsDetail %q -dir <path>\n", key)
		fmt.Println()
	}
	return nil
}

func (fmc *FrontMatterChecker) runListDateFormatsDetail(files []string) error {
	for _, key := range fmc.ListDateFormatsDetail {
		fmt.Printf("Date format detail for %q:\n", key)
		tbl := NewTable("File", "Format", "Length", "Value")
		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("warning: could not read %s: %v\n", file, err)
				continue
			}
			fm, err := frontmatter.GetFrontMatterMap(string(content))
			if err != nil || len(fm) == 0 {
				continue
			}
			val, ok := frontmatter.NestedGet(fm, frontmatter.KeyPath(key))
			if !ok {
				continue
			}
			s, isStr := val.(string)
			if !isStr {
				s = fmt.Sprintf("%v", val)
			}
			format := "(empty)"
			if strings.TrimSpace(s) != "" {
				if f := detectDateFormat(s); f != "" {
					format = f
				} else {
					format = "(unrecognized)"
				}
			}
			tbl.AddRow(displayPath(file, fmc.PathKeep), format, fmt.Sprintf("%d", len(s)), s)
		}
		tbl.Print()
		fmt.Println()
	}
	return nil
}

func (fmc *FrontMatterChecker) runListValues(files []string) error {
	for _, key := range fmc.ListValues {
		counts := make(map[string]int)
		missing := 0

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("warning: could not read %s: %v\n", file, err)
				continue
			}
			fm, err := frontmatter.GetFrontMatterMap(string(content))
			if err != nil || len(fm) == 0 {
				missing++
				continue
			}
			val, ok := frontmatter.NestedGet(fm, frontmatter.KeyPath(key))
			if !ok {
				missing++
				continue
			}
			counts[fmt.Sprintf("%v", val)]++
		}

		// sort by count descending, then value ascending for ties
		type entry struct {
			val   string
			count int
		}
		entries := make([]entry, 0, len(counts))
		for v, c := range counts {
			entries = append(entries, entry{v, c})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].count != entries[j].count {
				return entries[i].count > entries[j].count
			}
			return entries[i].val < entries[j].val
		})

		fmt.Printf("Values for %q:\n", key)
		for _, e := range entries {
			fmt.Printf("  %-40s %d\n", e.val, e.count)
		}
		if missing > 0 {
			fmt.Printf("  %-40s %d\n", "(missing)", missing)
		}
		fmt.Println()
	}
	return nil
}

func (fmc *FrontMatterChecker) runCheckTypes(files []string) error {
	type check struct {
		key      string
		typeName string
	}

	checks := make([]check, 0, len(fmc.CheckTypes))
	for _, entry := range fmc.CheckTypes {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -checkType %q: expected key:type (e.g. disable:bool)", entry)
		}
		supported := map[string]bool{"bool": true, "string": true, "int": true, "float": true, "list": true, "array": true, "map": true, "object": true}
		if !supported[parts[1]] {
			return fmt.Errorf("invalid -checkType %q: unknown type %q (supported: bool, string, int, float, list, map)", entry, parts[1])
		}
		checks = append(checks, check{key: parts[0], typeName: parts[1]})
	}

	for _, chk := range checks {
		fmt.Printf("Checking %s is type %s:\n", chk.key, chk.typeName)
		found := false
		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("  warning: could not read %s: %v\n", file, err)
				continue
			}
			fm, err := frontmatter.GetFrontMatterMap(string(content))
			if err != nil || len(fm) == 0 {
				continue
			}
			val, ok := frontmatter.NestedGet(fm, frontmatter.KeyPath(chk.key))
			if !ok {
				continue // absent — not a type violation
			}
			if !matchesType(val, chk.typeName) {
				fmt.Printf("  %s  (actual type: %s, value: %v)\n", displayPath(file, fmc.PathKeep), yamlTypeName(val), val)
				found = true
			}
		}
		if !found {
			fmt.Println("  all files conform")
		}
		fmt.Println()
	}
	return nil
}

func (fmc *FrontMatterChecker) runCheckFormats(files []string) error {
	type check struct {
		key    string
		format string
		layout string
	}

	checks := make([]check, 0, len(fmc.CheckFormats))
	for _, entry := range fmc.CheckFormats {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -checkFormat %q: expected key:FORMAT (e.g. last_update.date:YYYYMMDD or id:uuid)", entry)
		}
		checks = append(checks, check{
			key:    parts[0],
			format: parts[1],
			layout: userFormatToGoLayout(parts[1]),
		})
	}

	for _, chk := range checks {
		fmt.Printf("Checking %s against format %s:\n", chk.key, chk.format)
		found := false
		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("  warning: could not read %s: %v\n", file, err)
				continue
			}
			fm, err := frontmatter.GetFrontMatterMap(string(content))
			if err != nil || len(fm) == 0 {
				continue
			}
			val, ok := frontmatter.NestedGet(fm, frontmatter.KeyPath(chk.key))
			if !ok {
				continue // property absent — not a format violation
			}
			s, ok := val.(string)
			if !ok {
				fmt.Printf("  %s  (not a string: %T)\n", displayPath(file, fmc.PathKeep), val)
				found = true
				continue
			}
			if !validateFormat(chk.format, chk.layout, s) {
				fmt.Printf("  %s  (value: %q)\n", displayPath(file, fmc.PathKeep), s)
				found = true
			}
		}
		if !found {
			fmt.Println("  all files conform")
		}
		fmt.Println()
	}
	return nil
}

// csvFields splits a comma-separated string into trimmed, non-empty fields.
func csvFields(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isBoolTrue(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func isEmptyVal(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

func (fmc *FrontMatterChecker) loadConfig() error {
	file, err := os.Open(fmc.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&fmc.Config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

func (fmc *FrontMatterChecker) getFiles() ([]string, error) {
	var files []string

	if fmc.Dir != "" {
		err := filepath.Walk(fmc.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), ".md") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to traverse directory: %v", err)
		}
	}

	if len(fmc.Files) > 0 {
		files = append(files, fmc.Files...)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no markdown files to process")
	}

	return files, nil
}

func (fmc *FrontMatterChecker) analyzeFiles(files []string, template map[string]any, templateKeys []string) error {
	tbl := NewTable("File", "Placement", "Missing Props", "Extra Props", "Empty Props", "Order")

	total, noFM := 0, 0
	missingPropsCount, extraPropsCount, emptyPropsCount, outOfOrderCount := 0, 0, 0, 0
	for _, file := range files {
		analysis, err := frontmatter.AnalyzeFile(file, template, templateKeys)
		if err != nil {
			tbl.AddRow(displayPath(file, fmc.PathKeep), "error", "", "", "", err.Error())
			continue
		}
		total++
		if !analysis.HasFrontMatter {
			noFM++
		}
		if len(analysis.MissingProps) > 0 {
			missingPropsCount++
		}
		if len(analysis.ExtraProps) > 0 {
			extraPropsCount++
		}
		if len(analysis.EmptyProps) > 0 {
			emptyPropsCount++
		}
		if analysis.OutOfOrder {
			outOfOrderCount++
		}

		if fmc.IssuesOnly && !analysis.HasIssues() {
			continue
		}

		order := "-"
		if analysis.HasFrontMatter && len(templateKeys) > 0 && len(analysis.MissingProps) == 0 {
			if analysis.OutOfOrder {
				order = "out_of_order"
			} else {
				order = "ok"
			}
		}

		tbl.AddRow(
			displayPath(file, fmc.PathKeep),
			string(analysis.Placement.Status),
			joinOrDash(analysis.MissingProps),
			joinOrDash(analysis.ExtraProps),
			joinOrDash(analysis.EmptyProps),
			order,
		)
	}
	tbl.Print()

	fmt.Printf("\nFiles analyzed: %d\n\n", total)

	summary := NewTable("Analysis Item", "File Count")
	summary.AddRow("Missing front matter", fmt.Sprintf("%d", noFM))
	summary.AddRow("Missing properties from template", fmt.Sprintf("%d", missingPropsCount))
	summary.AddRow("Extra properties", fmt.Sprintf("%d", extraPropsCount))
	summary.AddRow("Properties with empty values", fmt.Sprintf("%d", emptyPropsCount))
	summary.AddRow("Properties not in template order", fmt.Sprintf("%d", outOfOrderCount))
	summary.Print()
	return nil
}

func (fmc *FrontMatterChecker) auditPlacement(files []string) error {
	results, err := frontmatter.AuditPlacementFiles(files)
	if err != nil {
		return err
	}

	tbl := NewTable("File", "Placement", "Reason", "Candidate Start Line")
	for _, result := range results {
		startLine := ""
		if result.Candidate != nil {
			startLine = fmt.Sprintf("%d", result.Candidate.StartLine)
		}
		tbl.AddRow(displayPath(result.FilePath, fmc.PathKeep), string(result.Status), result.Reason, startLine)
	}
	tbl.Print()
	return nil
}

func applyPlans(plans []frontmatter.FileChangePlan) error {
	if len(plans) == 0 {
		fmt.Println("No changes needed.")
		return nil
	}

	fmt.Println("Planned changes:")
	fmt.Println()
	for _, plan := range plans {
		fmt.Printf("  %s\n", plan.FilePath)
		for _, change := range plan.Changes {
			if change.RenamedFrom != "" {
				fmt.Printf("    %-20s %q → %q (renamed from %q)\n", change.Key+":", change.RenamedFrom, change.Key, change.RenamedFrom)
			} else {
				oldStr := fmt.Sprintf("%v", change.OldValue)
				if change.OldValue == nil {
					oldStr = "<missing>"
				}
				fmt.Printf("    %-20s %s → %v\n", change.Key+":", oldStr, change.NewValue)
			}
		}
		for _, key := range plan.KeysToDelete {
			fmt.Printf("    [delete] %s\n", key)
		}
		fmt.Println()
	}

	fmt.Print("Apply these changes? [Y/n]: ")
	var response string
	fmt.Scanln(&response)
	if response != "" && strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	for _, plan := range plans {
		if err := frontmatter.ApplyChangePlan(plan); err != nil {
			fmt.Printf("error: %s: %v\n", plan.FilePath, err)
		} else {
			fmt.Printf("updated: %s\n", plan.FilePath)
		}
	}

	return nil
}

func (fmc *FrontMatterChecker) fixFiles(files []string, template map[string]any, policies []frontmatter.PropertyPolicy) error {
	var plans []frontmatter.FileChangePlan

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		plan, err := frontmatter.PlanChanges(file, string(content), template, policies)
		if err != nil {
			fmt.Printf("warning: could not plan changes for %s: %v\n", file, err)
			continue
		}
		if plan.HasChanges() {
			plans = append(plans, plan)
		}
	}

	return applyPlans(plans)
}

func (fmc *FrontMatterChecker) setValues(files []string) error {
	template := map[string]any{}
	policies := make([]frontmatter.PropertyPolicy, 0, len(fmc.SetValues))

	for _, entry := range fmc.SetValues {
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -setValue %q: expected key:source:value[:action]", entry)
		}
		key, source, rest := parts[0], parts[1], parts[2]

		action := frontmatter.ActionAddIfMissing
		value := rest
		switch {
		case strings.HasSuffix(rest, ":always"):
			action = frontmatter.ActionOverwriteAlways
			value = strings.TrimSuffix(rest, ":always")
		case strings.HasSuffix(rest, ":if_empty"):
			action = frontmatter.ActionOverwriteIfEmpty
			value = strings.TrimSuffix(rest, ":if_empty")
		}

		// optional type specifier on static values: key:static:val:bool[:action]
		var staticTypeName string
		for _, t := range []string{"bool", "string", "int", "float"} {
			if strings.HasSuffix(value, ":"+t) {
				staticTypeName = t
				value = strings.TrimSuffix(value, ":"+t)
				break
			}
		}

		policy := frontmatter.PropertyPolicy{
			Key:    key,
			Action: action,
			Source: frontmatter.ValueSource(source),
		}
		switch frontmatter.ValueSource(source) {
		case frontmatter.SourceStatic:
			if staticTypeName != "" {
				casted, err := castValue(value, staticTypeName)
				if err != nil {
					return fmt.Errorf("-setValue %q: %v", entry, err)
				}
				policy.StaticValue = casted
			} else {
				policy.StaticValue = parseStaticValue(value)
			}
		case frontmatter.SourceComputed, frontmatter.SourceLLM:
			policy.Fn = value
		case frontmatter.SourceTransform:
			// value is "fn:from_key" after action suffix has been stripped
			tparts := strings.SplitN(value, ":", 2)
			if len(tparts) != 2 || tparts[0] == "" || tparts[1] == "" {
				return fmt.Errorf("transform setValue requires fn:from_key in %q", entry)
			}
			policy.Fn = tparts[0]
			policy.FromKey = tparts[1]
		default:
			return fmt.Errorf("invalid source %q in -setValue %q: expected static|computed|transform|llm", source, entry)
		}

		template[key] = ""
		policies = append(policies, policy)
	}

	return fmc.fixFiles(files, template, policies)
}

func (fmc *FrontMatterChecker) createFrontMatter(files []string, template map[string]any) error {
	defaults := make(map[string]any, len(fmc.FmDefaults))
	for _, entry := range fmc.FmDefaults {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			return fmt.Errorf("invalid -fmDefault %q: expected key:value", entry)
		}
		defaults[parts[0]] = parts[1]
	}

	targetStatus := frontmatter.PlacementMissing
	if fmc.OnManualReview {
		targetStatus = frontmatter.PlacementManualReview
	}

	var plans []frontmatter.FrontMatterCreationPlan
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		plan, err := frontmatter.PlanFrontMatterCreation(file, string(content), template, defaults, 5, targetStatus)
		if err != nil {
			fmt.Printf("warning: %s: %v\n", file, err)
			continue
		}
		if plan.ShouldCreate() {
			plans = append(plans, plan)
		}
	}

	if len(plans) == 0 {
		fmt.Println("No files need front matter creation.")
		return nil
	}

	// Warn about keys that will be blank and suggest follow-up commands.
	var blankKeys []string
	for k := range template {
		if _, hasDefault := defaults[k]; !hasDefault {
			blankKeys = append(blankKeys, k)
		}
	}
	sort.Strings(blankKeys)
	if len(blankKeys) > 0 {
		fmt.Printf("Note: %d key(s) have no -fmDefault and will be added blank: %s\n",
			len(blankKeys), strings.Join(blankKeys, ", "))
		suggestions := buildPostCreateSuggestions(blankKeys, fmc.Dir, fmc.Files)
		if len(suggestions) > 0 {
			fmt.Println("  After creation, consider running:")
			for _, s := range suggestions {
				fmt.Printf("    %s\n", s)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Will add front matter to %d file(s):\n", len(plans))
	for _, plan := range plans {
		fmt.Printf("\n  %s\n", plan.FilePath)
		for _, line := range plan.Preview {
			fmt.Printf("    %s\n", line)
		}
	}

	fmt.Print("\nApply these changes? [Y/n]: ")
	var response string
	fmt.Scanln(&response)
	if response != "" && strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	for _, plan := range plans {
		if err := frontmatter.ApplyFrontMatterCreation(plan); err != nil {
			fmt.Printf("error: %s: %v\n", plan.FilePath, err)
		} else {
			fmt.Printf("  wrote %s\n", plan.FilePath)
		}
	}
	return nil
}

// buildPostCreateSuggestions returns example fmc commands to populate blank
// keys after front matter has been created. It recognises common naming
// patterns (id → uuid, date/updated/modified → today).
func buildPostCreateSuggestions(blankKeys []string, dir string, files []string) []string {
	// Build the target part of the command ("-dir <dir>" or "-files <f1,f2>").
	target := ""
	if dir != "" {
		target = fmt.Sprintf(" -dir %s", dir)
	} else if len(files) > 0 {
		target = fmt.Sprintf(" -files %s", strings.Join(files, ","))
	}

	datePatterns := []string{"date", "updated", "modified", "last_update", "created"}
	isDateLike := func(k string) bool {
		kl := strings.ToLower(k)
		for _, p := range datePatterns {
			if kl == p || strings.Contains(kl, p) {
				return true
			}
		}
		return false
	}

	var suggestions []string
	for _, k := range blankKeys {
		kl := strings.ToLower(k)
		switch {
		case kl == "id":
			suggestions = append(suggestions, fmt.Sprintf("fmc -setValue id:computed:uuid%s", target))
		case isDateLike(k):
			// Suggest the nested path if the key looks like a parent struct
			// (e.g. last_update → last_update.date).
			if kl == "last_update" || kl == "lastupdated" {
				suggestions = append(suggestions, fmt.Sprintf("fmc -setValue %s.date:computed:today%s", k, target))
			} else {
				suggestions = append(suggestions, fmt.Sprintf("fmc -setValue %s:computed:today%s", k, target))
			}
		case kl == "slug" || kl == "url_slug":
			suggestions = append(suggestions, fmt.Sprintf("fmc -setValue %s:transform:slug:title%s", k, target))
		}
	}
	return suggestions
}

func (fmc *FrontMatterChecker) addMissingProps(files []string, template map[string]any) error {
	policies := make([]frontmatter.PropertyPolicy, 0, len(template))
	for key := range template {
		policies = append(policies, frontmatter.PropertyPolicy{
			Key:    key,
			Action: frontmatter.ActionAddIfMissing,
			Source: frontmatter.SourceStatic,
		})
	}
	return fmc.fixFiles(files, template, policies)
}

// displayPath trims a file path to the last (keep+1) segments.
// keep < 0 returns the full path; keep == 0 returns only the filename.
func displayPath(path string, keep int) string {
	if keep < 0 {
		return path
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	take := keep + 1
	if take >= len(nonEmpty) {
		return path
	}
	return "<hidden>/" + strings.Join(nonEmpty[len(nonEmpty)-take:], "/")
}

func (fmc *FrontMatterChecker) inspectProps(files []string) error {
	for _, propKey := range fmc.InspectProps {
		fmt.Printf("## Property: %s\n\n", propKey)

		type nodeStats struct {
			depths    map[int]bool
			fileCount int
		}
		summaryStats := map[string]*nodeStats{}

		tbl := NewTable("File", "Present", "Max Depth", "Sub-properties")
		for _, file := range files {
			label := displayPath(file, fmc.PathKeep)
			content, err := os.ReadFile(file)
			if err != nil {
				tbl.AddRow(label, "error", "-", err.Error())
				continue
			}
			insp, err := frontmatter.InspectProperty(string(content), propKey)
			if err != nil {
				tbl.AddRow(label, "error", "-", err.Error())
				continue
			}
			if !insp.Present {
				tbl.AddRow(label, "no", "-", "-")
				continue
			}
			subKeys := make([]string, 0, len(insp.Nodes))
			seen := map[string]bool{}
			for _, n := range insp.Nodes {
				if !seen[n.Key] {
					subKeys = append(subKeys, n.Key)
					seen[n.Key] = true
				}
				if _, ok := summaryStats[n.Key]; !ok {
					summaryStats[n.Key] = &nodeStats{depths: map[int]bool{}}
				}
				summaryStats[n.Key].depths[n.Depth] = true
			}
			for _, k := range subKeys {
				summaryStats[k].fileCount++
			}
			depthStr := "-"
			if !insp.IsScalar {
				depthStr = fmt.Sprintf("%d", insp.MaxDepth)
			}
			subStr := "-"
			if len(subKeys) > 0 {
				subStr = strings.Join(subKeys, ", ")
			}
			tbl.AddRow(label, "yes", depthStr, subStr)
		}
		tbl.Print()

		if len(summaryStats) > 0 {
			fmt.Printf("\n### Summary\n\n")
			keys := make([]string, 0, len(summaryStats))
			for k := range summaryStats {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			sumTbl := NewTable("Sub-property", "Depths", "File Count")
			for _, k := range keys {
				st := summaryStats[k]
				depths := make([]int, 0, len(st.depths))
				for d := range st.depths {
					depths = append(depths, d)
				}
				sort.Ints(depths)
				depthStrs := make([]string, 0, len(depths))
				for _, d := range depths {
					depthStrs = append(depthStrs, fmt.Sprintf("%d", d))
				}
				sumTbl.AddRow(k, strings.Join(depthStrs, ", "), fmt.Sprintf("%d", st.fileCount))
			}
			sumTbl.Print()
		}
		fmt.Println()
	}
	return nil
}

func printRankedSummary(counts map[string]int) {
	type kv struct{ Key string; Count int }
	ranked := make([]kv, 0, len(counts))
	for k, v := range counts {
		ranked = append(ranked, kv{k, v})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Count != ranked[j].Count {
			return ranked[i].Count > ranked[j].Count
		}
		return ranked[i].Key < ranked[j].Key
	})
	tbl := NewTable("Property", "Count")
	for _, e := range ranked {
		tbl.AddRow(e.Key, fmt.Sprintf("%d", e.Count))
	}
	tbl.Print()
}

// listEmptyForKey reports files where the specified keys exist but are empty.
func (fmc *FrontMatterChecker) listEmptyForKey(files []string) error {
	keys := []string(fmc.ListEmptyForKey)
	tbl := NewTable("File", "Empty Props")
	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			tbl.AddRow(displayPath(file, fmc.PathKeep), "error: "+err.Error())
			continue
		}
		empty, err := frontmatter.FindEmptyProps(string(content), keys)
		if err != nil {
			continue
		}
		if len(empty) == 0 {
			continue
		}
		tbl.AddRow(displayPath(file, fmc.PathKeep), strings.Join(empty, ", "))
		for _, k := range empty {
			counts[k]++
		}
	}
	tbl.Print()

	if len(counts) == 0 {
		fmt.Println("\nNo empty properties found.")
		return nil
	}
	fmt.Println("\nSummary:")
	printRankedSummary(counts)
	return nil
}

// listEmptyAll scans every key in each file and shows a ranked counts table
// of which properties are most frequently empty.
func (fmc *FrontMatterChecker) listEmptyAll(files []string) error {
	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil || fm == nil {
			continue
		}
		keys := make([]string, 0, len(fm))
		for k := range fm {
			keys = append(keys, k)
		}
		empty, err := frontmatter.FindEmptyProps(string(content), keys)
		if err != nil {
			continue
		}
		for _, k := range empty {
			counts[k]++
		}
	}

	if len(counts) == 0 {
		fmt.Println("No empty properties found.")
		return nil
	}
	fmt.Printf("Empty property counts across %d files:\n\n", len(files))
	printRankedSummary(counts)
	return nil
}

type fileEmptyResult struct {
	path  string
	count int
	keys  []string
}

// listEmptyDetails shows a per-file breakdown: file | # empty | empty keys.
// Sortable by "name" or "count" via -sortBy (default: count desc).
func (fmc *FrontMatterChecker) listEmptyDetails(files []string) error {
	var results []fileEmptyResult

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil || fm == nil {
			continue
		}
		keys := make([]string, 0, len(fm))
		for k := range fm {
			keys = append(keys, k)
		}
		empty, err := frontmatter.FindEmptyProps(string(content), keys)
		if err != nil || len(empty) == 0 {
			continue
		}
		sort.Strings(empty)
		results = append(results, fileEmptyResult{
			path:  displayPath(file, fmc.PathKeep),
			count: len(empty),
			keys:  empty,
		})
	}

	if len(results) == 0 {
		fmt.Println("No empty properties found.")
		return nil
	}

	switch fmc.SortBy {
	case "name":
		sort.Slice(results, func(i, j int) bool {
			return results[i].path < results[j].path
		})
	default: // "count" — descending
		sort.Slice(results, func(i, j int) bool {
			if results[i].count != results[j].count {
				return results[i].count > results[j].count
			}
			return results[i].path < results[j].path
		})
	}

	tbl := NewTable("File", "# Empty", "Empty Keys")
	for _, r := range results {
		tbl.AddRow(r.path, fmt.Sprintf("%d", r.count), strings.Join(r.keys, ", "))
	}
	tbl.Print()
	fmt.Printf("\n%d file(s) with empty properties.\n", len(results))
	return nil
}

func (fmc *FrontMatterChecker) removeEmpty(files []string) error {
	var plans []frontmatter.FileChangePlan

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		var keys []string
		if fmc.RemoveEmpty == "all" {
			fm, ferr := frontmatter.GetFrontMatterMap(string(content))
			if ferr != nil || fm == nil {
				continue
			}
			for k := range fm {
				keys = append(keys, k)
			}
		} else {
			keys = csvFields(fmc.RemoveEmpty)
		}
		plan, err := frontmatter.PlanRemoveIfEmpty(file, string(content), keys)
		if err != nil {
			fmt.Printf("warning: could not plan for %s: %v\n", file, err)
			continue
		}
		if plan.HasChanges() {
			plans = append(plans, plan)
		}
	}

	return applyPlans(plans)
}

func (fmc *FrontMatterChecker) removeExtraProps(files []string, template map[string]any) error {
	var plans []frontmatter.FileChangePlan

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		plan, err := frontmatter.PlanRemoveExtraProps(file, string(content), template)
		if err != nil {
			fmt.Printf("warning: could not plan for %s: %v\n", file, err)
			continue
		}
		if plan.HasChanges() {
			plans = append(plans, plan)
		}
	}

	return applyPlans(plans)
}

func (fmc *FrontMatterChecker) listExtraProps(files []string, template map[string]any) error {
	tbl := NewTable("File", "Extra Props")
	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			tbl.AddRow(displayPath(file, fmc.PathKeep), "error: "+err.Error())
			continue
		}
		extras, err := frontmatter.FindExtraProps(string(content), template)
		if err != nil {
			tbl.AddRow(displayPath(file, fmc.PathKeep), "error: "+err.Error())
			continue
		}
		tbl.AddRow(displayPath(file, fmc.PathKeep), joinOrDash(extras))
		for _, k := range extras {
			counts[k]++
		}
	}
	tbl.Print()

	if len(counts) == 0 {
		fmt.Println("\nNo extra properties found.")
		return nil
	}
	fmt.Println("\nSummary:")
	printRankedSummary(counts)
	return nil
}

func (fmc *FrontMatterChecker) listMissingProps(files []string, template map[string]any) error {
	tbl := NewTable("File", "Missing Props")
	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			tbl.AddRow(displayPath(file, fmc.PathKeep), "error: "+err.Error())
			continue
		}
		missing, err := frontmatter.FindMissingProps(string(content), template)
		if err != nil {
			tbl.AddRow(displayPath(file, fmc.PathKeep), "error: "+err.Error())
			continue
		}
		tbl.AddRow(displayPath(file, fmc.PathKeep), joinOrDash(missing))
		for _, k := range missing {
			counts[k]++
		}
	}
	tbl.Print()

	if len(counts) == 0 {
		fmt.Println("\nNo missing properties found.")
		return nil
	}
	fmt.Println("\nSummary:")
	printRankedSummary(counts)
	return nil
}

func (fmc *FrontMatterChecker) runReorder(files []string) error {
	firstKeys := csvFields(fmc.KeysToTop)
	lastKeys := csvFields(fmc.KeysToBottom)

	var plans []frontmatter.ReorderPlan
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		plan, err := frontmatter.PlanReorder(file, string(content), firstKeys, lastKeys)
		if err != nil {
			// no front matter — skip silently
			continue
		}
		if plan.HasChange || len(plan.MissingKeys) > 0 {
			plans = append(plans, plan)
		}
	}

	if len(plans) == 0 {
		fmt.Println("No files need reordering.")
		return nil
	}

	actionCount := 0
	for _, p := range plans {
		fmt.Printf("  %s\n", displayPath(p.FilePath, fmc.PathKeep))
		if p.HasChange {
			fmt.Printf("    %s\n    → %s\n", strings.Join(p.OldOrder, ", "), strings.Join(p.NewOrder, ", "))
			actionCount++
		} else {
			fmt.Printf("    (order unchanged)\n")
		}
		if len(p.MissingKeys) > 0 {
			fmt.Printf("    not found (will not be created): %s\n", strings.Join(p.MissingKeys, ", "))
		}
	}

	if actionCount == 0 {
		fmt.Println("\nNo order changes to apply (all listed keys were missing).")
		return nil
	}

	fmt.Printf("\nApply reorder to %d file(s)? [Y/n]: ", actionCount)
	var response string
	fmt.Scanln(&response)
	if response != "" && strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	for _, plan := range plans {
		if !plan.HasChange {
			continue
		}
		if err := frontmatter.ApplyReorder(plan); err != nil {
			fmt.Printf("error: %s: %v\n", plan.FilePath, err)
		} else {
			fmt.Printf("  wrote %s\n", displayPath(plan.FilePath, fmc.PathKeep))
		}
	}
	return nil
}

func (fmc *FrontMatterChecker) runGenID(files []string) error {
	type filePlan struct {
		file   string
		policy frontmatter.PropertyPolicy
		reason string // "missing/empty" or "invalid UUID"
	}

	var plans []filePlan
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil {
			fmt.Printf("warning: %s: %v\n", file, err)
			continue
		}

		idVal, exists := fm["id"]
		idStr, _ := idVal.(string)

		var reason string
		var action frontmatter.PropertyAction
		switch {
		case !exists || strings.TrimSpace(idStr) == "":
			reason = "missing or empty"
			action = frontmatter.ActionOverwriteIfEmpty
		case fmc.GenIDOverwriteInvalid && !reUUID.MatchString(idStr):
			reason = fmt.Sprintf("invalid UUID (current: %q)", idStr)
			action = frontmatter.ActionOverwriteAlways
		default:
			continue
		}

		plans = append(plans, filePlan{
			file: file,
			policy: frontmatter.PropertyPolicy{
				Key:    "id",
				Action: action,
				Source: frontmatter.SourceComputed,
				Fn:     "uuid",
			},
			reason: reason,
		})
	}

	if len(plans) == 0 {
		fmt.Println("No files need an ID.")
		return nil
	}

	fmt.Printf("Will set id on %d file(s):\n", len(plans))
	for _, p := range plans {
		fmt.Printf("  %s  (%s)\n", displayPath(p.file, fmc.PathKeep), p.reason)
	}

	modifiedFiles := make([]string, len(plans))
	for i, p := range plans {
		modifiedFiles[i] = p.file
	}
	warnPotentialBrokenLinks(modifiedFiles, files)

	fmt.Print("Apply these changes? [Y/n]: ")
	var response string
	fmt.Scanln(&response)
	if response != "" && strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	for _, p := range plans {
		content, err := os.ReadFile(p.file)
		if err != nil {
			fmt.Printf("error: %s: %v\n", p.file, err)
			continue
		}
		changePlan, err := frontmatter.PlanChanges(p.file, string(content), map[string]any{"id": ""}, []frontmatter.PropertyPolicy{p.policy})
		if err != nil {
			fmt.Printf("error: %s: %v\n", p.file, err)
			continue
		}
		if err := frontmatter.ApplyChangePlan(changePlan); err != nil {
			fmt.Printf("error: %s: %v\n", p.file, err)
		} else {
			fmt.Printf("  wrote %s\n", displayPath(p.file, fmc.PathKeep))
		}
	}
	return nil
}

// warnPotentialBrokenLinks searches all files for lines that look like links
// to any of the target files (by filename stem or current id/slug value).
// It prints a warning and a table of candidates before destructive operations
// that change id or slug. The search is heuristic — not authoritative.
func warnPotentialBrokenLinks(targetFiles []string, allFiles []string) {
	type hit struct {
		linkFile string
		line     int
		text     string
		term     string
	}

	// Build a map of search terms → originating target file.
	type termInfo struct {
		targetFile string
		kind       string // "filename" or "id" or "slug"
	}
	terms := make(map[string]termInfo)
	for _, f := range targetFiles {
		stem := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		if stem != "" {
			terms[stem] = termInfo{f, "filename"}
		}
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil || fm == nil {
			continue
		}
		for _, key := range []string{"id", "slug"} {
			if v, ok := fm[key]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					terms[s] = termInfo{f, key}
				}
			}
		}
	}

	if len(terms) == 0 {
		return
	}

	var hits []hit
	targetSet := make(map[string]bool, len(targetFiles))
	for _, f := range targetFiles {
		targetSet[f] = true
	}

	for _, f := range allFiles {
		if targetSet[f] {
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for term, info := range terms {
				if strings.Contains(line, term) {
					hits = append(hits, hit{
						linkFile: f,
						line:     i + 1,
						text:     strings.TrimSpace(line),
						term:     fmt.Sprintf("%s (%s in %s)", term, info.kind, filepath.Base(info.targetFile)),
					})
					break // one hit per line
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("WARNING: Changing 'id' or 'slug' will break any existing links to these files.")
	fmt.Println("After making changes, run 'npx docusaurus build' to find broken links.")
	fmt.Println("Fix them with find-and-replace (time-consuming but straightforward).")
	fmt.Println()

	if len(hits) == 0 {
		fmt.Println("No potential links to the affected files were found.")
		fmt.Println()
		return
	}

	fmt.Printf("Potential links found (%d) — not authoritative, verify before proceeding:\n\n", len(hits))
	tbl := NewTable("File", "Line", "Matched Term", "Content")
	for _, h := range hits {
		preview := h.text
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		tbl.AddRow(h.linkFile, fmt.Sprintf("%d", h.line), h.term, preview)
	}
	tbl.Print()
	fmt.Println()
}

func (fmc *FrontMatterChecker) replaceKeys(files []string) error {
	// Build a minimal template and rename policies from -replaceKey OldKey:NewKey entries
	template := map[string]any{}
	policies := make([]frontmatter.PropertyPolicy, 0, len(fmc.ReplaceKeys))

	for _, entry := range fmc.ReplaceKeys {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -replaceKey value %q: expected OldKey:NewKey", entry)
		}
		oldKey, newKey := parts[0], parts[1]
		template[newKey] = ""
		policies = append(policies, frontmatter.PropertyPolicy{
			Key:     newKey,
			Action:  frontmatter.ActionRenameFrom,
			FromKey: oldKey,
		})
	}

	return fmc.fixFiles(files, template, policies)
}

func (fmc *FrontMatterChecker) createFrom(files []string) error {
	template := map[string]any{}
	policies := make([]frontmatter.PropertyPolicy, 0, len(fmc.CreateFrom))

	for _, entry := range fmc.CreateFrom {
		// Format: from:to[:action][:transform:fn]
		// Split on ":" but we need to find the optional "transform" segment.
		// We consume parts left-to-right: from, to, then optional action keyword,
		// then optional literal "transform" followed by fn name.
		parts := strings.Split(entry, ":")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -createFrom value %q: expected from:to[:action][:transform:fn]", entry)
		}
		fromKey, toKey := parts[0], parts[1]
		rest := parts[2:]

		action := frontmatter.ActionAddIfMissing
		fn := "copy"

		// consume optional action keyword
		if len(rest) > 0 {
			switch rest[0] {
			case "always":
				action = frontmatter.ActionOverwriteAlways
				rest = rest[1:]
			case "if_empty":
				action = frontmatter.ActionOverwriteIfEmpty
				rest = rest[1:]
			case "add_if_missing":
				rest = rest[1:]
			case "transform":
				// no action specified, transform comes first
			default:
				return fmt.Errorf("invalid action %q in -createFrom %q: expected always|if_empty|add_if_missing", rest[0], entry)
			}
		}

		// consume optional transform:fn
		if len(rest) >= 2 && rest[0] == "transform" {
			fn = rest[1]
		} else if len(rest) == 1 && rest[0] == "transform" {
			return fmt.Errorf("invalid -createFrom value %q: 'transform' must be followed by a function name", entry)
		} else if len(rest) > 0 {
			return fmt.Errorf("invalid -createFrom value %q: unexpected segment %q", entry, rest[0])
		}

		template[toKey] = ""
		policies = append(policies, frontmatter.PropertyPolicy{
			Key:     toKey,
			Action:  action,
			Source:  frontmatter.SourceTransform,
			Fn:      fn,
			FromKey: fromKey,
		})
	}

	// Warn about potential broken links when writing to id or slug.
	touchesLinkKey := false
	for _, p := range policies {
		if p.Key == "id" || p.Key == "slug" {
			touchesLinkKey = true
			break
		}
	}
	if touchesLinkKey {
		warnPotentialBrokenLinks(files, files)
	}

	return fmc.fixFiles(files, template, policies)
}

// runGenerateSources populates tag_sources and keyword_sources from the named
// source. Currently only "filepath" is supported.
func (fmc *FrontMatterChecker) runGenerateSources(files []string) error {
	switch fmc.GenerateSources {
	case "filepath":
		return fmc.generateSourcesFilepath(files)
	default:
		return fmt.Errorf("unknown source %q: only 'filepath' is currently supported", fmc.GenerateSources)
	}
}

func (fmc *FrontMatterChecker) generateSourcesFilepath(files []string) error {
	today := time.Now().Format("2006-01-02")
	var plans []frontmatter.FileChangePlan

	for _, file := range files {
		segments := frontmatter.ExtractPathSegments(file, 0)
		if len(segments) == 0 {
			continue
		}
		// Convert []string to []any for YAML marshaling consistency.
		segsAny := make([]any, len(segments))
		for i, s := range segments {
			segsAny[i] = s
		}
		plan := frontmatter.FileChangePlan{FilePath: file}
		plan.Changes = []frontmatter.PropChange{
			{Key: "tag_sources.filepath.date_last_generated", NewValue: today},
			{Key: "tag_sources.filepath.tag_list", NewValue: segsAny},
			{Key: "keyword_sources.filepath.date_last_generated", NewValue: today},
			{Key: "keyword_sources.filepath.keyword_list", NewValue: segsAny},
		}
		plans = append(plans, plan)
	}

	return applyPlans(plans)
}

// runRollup merges staged source lists into the top-level tags/keywords fields.
func (fmc *FrontMatterChecker) runRollup(files []string) error {
	props := csvFields(fmc.Rollup)
	if len(props) == 0 {
		return fmt.Errorf("-rollup requires a value: tags, keywords, or tags,keywords")
	}
	sources := csvFields(fmc.RollupSources)
	if len(sources) == 0 {
		return fmt.Errorf("-rollupSources is required when using -rollup")
	}

	for _, p := range props {
		if p != "tags" && p != "keywords" {
			return fmt.Errorf("-rollup value %q not recognised: use tags, keywords, or tags,keywords", p)
		}
	}

	var plans []frontmatter.FileChangePlan

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", file, err)
			continue
		}
		fm, err := frontmatter.GetFrontMatterMap(string(content))
		if err != nil || fm == nil {
			continue
		}

		plan := frontmatter.FileChangePlan{FilePath: file}

		for _, prop := range props {
			sourcesProp, listKey := rollupPropNames(prop)

			sourcesMap, _ := fm[sourcesProp].(map[string]any)
			union := collectSourceUnion(sourcesMap, sources, listKey)
			if len(union) == 0 {
				continue
			}

			existing := frontmatter.ToStringSlice(fm[prop])
			var newVal []string
			if fmc.RollupNoPreserve {
				newVal = union
			} else {
				newVal = stringUnion(existing, union)
			}

			if stringSlicesEqualUnordered(existing, newVal) {
				continue
			}

			// Build old/new as []any for display and YAML consistency.
			oldAny := stringSliceToAny(existing)
			newAny := stringSliceToAny(newVal)

			// If noPreserve, annotate removed items in the plan display.
			if fmc.RollupNoPreserve {
				removed := stringSubtract(existing, newVal)
				if len(removed) > 0 {
					fmt.Printf("note: %s — removing from %s: %v\n", displayPath(file, fmc.PathKeep), prop, removed)
				}
			}

			plan.Changes = append(plan.Changes, frontmatter.PropChange{
				Key:      prop,
				OldValue: oldAny,
				NewValue: newAny,
			})
		}

		if plan.HasChanges() {
			plans = append(plans, plan)
		}
	}

	return applyPlans(plans)
}

// rollupPropNames returns the sources property name and list key for a given
// top-level property ("tags" → "tag_sources", "tag_list").
func rollupPropNames(prop string) (sourcesProp, listKey string) {
	switch prop {
	case "tags":
		return "tag_sources", "tag_list"
	case "keywords":
		return "keyword_sources", "keyword_list"
	}
	return prop + "_sources", prop[:len(prop)-1] + "_list"
}

// collectSourceUnion gathers the union of all tag/keyword lists from the
// selected sources within a sources map. Source names use dot notation for
// nested sources (e.g. "llm.gpt-4o"). "all" walks the entire tree.
func collectSourceUnion(sourcesMap map[string]any, sourceNames []string, listKey string) []string {
	if sourcesMap == nil {
		return nil
	}
	seen := map[string]bool{}
	var result []string

	addList := func(m map[string]any) {
		for _, v := range frontmatter.ToStringSlice(m[listKey]) {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		}
	}

	if len(sourceNames) == 1 && sourceNames[0] == "all" {
		walkSourceMap(sourcesMap, listKey, addList)
		return result
	}

	for _, name := range sourceNames {
		path := strings.Split(name, ".")
		val, ok := frontmatter.NestedGet(sourcesMap, path)
		if !ok {
			continue
		}
		m, ok := val.(map[string]any)
		if !ok {
			continue
		}
		addList(m)
	}
	return result
}

// walkSourceMap recursively visits every leaf source map (one that contains
// listKey) and calls fn on it.
func walkSourceMap(m map[string]any, listKey string, fn func(map[string]any)) {
	if _, hasListKey := m[listKey]; hasListKey {
		fn(m)
		return
	}
	for _, v := range m {
		if child, ok := v.(map[string]any); ok {
			walkSourceMap(child, listKey, fn)
		}
	}
}

// stringUnion returns the union of a and b preserving order (a first, then new
// items from b).
func stringUnion(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	result := append([]string{}, a...)
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// stringSubtract returns elements in a that are not in b.
func stringSubtract(a, b []string) []string {
	bSet := make(map[string]bool, len(b))
	for _, s := range b {
		bSet[s] = true
	}
	var out []string
	for _, s := range a {
		if !bSet[s] {
			out = append(out, s)
		}
	}
	return out
}

// stringSlicesEqualUnordered returns true if both slices have identical
// elements regardless of order.
func stringSlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}

func stringSliceToAny(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}

func valueOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
