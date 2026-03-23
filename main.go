package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	GenID              bool
	Config             Config

	IssuesOnly       bool
	Verbose          bool
	ListExtraProps   bool
	ListMissingProps bool
	ReplaceKeys      repeatableFlag // each entry: "OldKey:NewKey"
	CreateSlugs      repeatableFlag // each entry: "FromKey:ToKey[:action]"
	SetValues        repeatableFlag // each entry: "key:source:value[:action]"
	AddMissingProps  bool
	RemoveExtraProps bool
	RemoveEmpty          repeatableFlag // each entry: property name
	ListEmpty            repeatableFlag // each entry: property name
	InspectProps         repeatableFlag // each entry: property name
	PathKeep             int            // -1 = full path, 0 = filename only, N = last N dirs + filename
	CreateFrontMatter    bool
	OnManualReview       bool
	FmDefaults           repeatableFlag // each entry: "key:value"
	AnalyzeOrder         bool
	AnalyzeSEO           bool
	Plugin               string // "docs" or "blog"
}

func main() {
	// policy subcommand intercepted before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "policy" {
		runPolicyCommand(os.Args[2:])
		return
	}

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

	///// Make Changes to Front Matter ///////
	//Single Property CRUD
	genID := flag.Bool("genID", false, "Generate IDs for files where the ID property is missing or empty")
	flag.Var(&checker.ReplaceKeys, "replaceKey", "Rename a key, keeping its value (repeatable; see: fmc help replaceKey)")
	flag.Var(&checker.CreateSlugs, "createSlug", "Create a URL slug from a property (repeatable; see: fmc help createSlug)")
	flag.Var(&checker.SetValues, "setValue", "Set a property via static, computed, or llm source (repeatable; see: fmc help setValue)")
	flag.Var(&checker.RemoveEmpty, "removeEmpty", "Remove a property if its value is empty or missing (repeatable)")
	flag.Var(&checker.ListEmpty, "listEmpty", "List files where a property exists but is empty or whitespace (repeatable)")
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
	checker.CreateFrontMatter = *createFrontMatter
	checker.OnManualReview = *onManualReview
	checker.AnalyzeOrder = *analyzeOrder
	checker.AnalyzeSEO = *analyzeSEO
	checker.Plugin = *plugin
	checker.ListExtraProps = *listExtraProps
	checker.ListMissingProps = *listMissingProps
	checker.AddMissingProps = *addMissingProps
	checker.RemoveExtraProps = *removeExtraProps

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

	if len(fmc.ReplaceKeys) > 0 {
		return fmc.replaceKeys(filesToProcess)
	}

	if len(fmc.CreateSlugs) > 0 {
		return fmc.createSlugs(filesToProcess)
	}

	if len(fmc.SetValues) > 0 {
		return fmc.setValues(filesToProcess)
	}

	if len(fmc.RemoveEmpty) > 0 {
		return fmc.removeEmpty(filesToProcess)
	}

	if len(fmc.ListEmpty) > 0 {
		return fmc.listEmpty(filesToProcess)
	}

	if fmc.AnalyzeSEO {
		if fmc.Plugin == "" {
			return fmt.Errorf("-analyzeSEO requires -plugin (docs or blog)")
		}
		return fmc.analyzeSEO(filesToProcess)
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

	fmt.Println("| File | Order |")
	fmt.Println("|---|---|")

	inOrder, outOfOrder, excluded := 0, 0, 0
	for _, r := range results {
		switch r.status {
		case "excluded":
			excluded++
			if !fmc.IssuesOnly {
				fmt.Printf("| %s | excluded |\n", displayPath(r.path, fmc.PathKeep))
			}
		case "ok":
			inOrder++
			if !fmc.IssuesOnly {
				fmt.Printf("| %s | ok |\n", displayPath(r.path, fmc.PathKeep))
			}
		case "out_of_order":
			outOfOrder++
			fmt.Printf("| %s | out_of_order |\n", displayPath(r.path, fmc.PathKeep))
		}
	}

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
	fmt.Println("| SEO Property | Missing | Empty |")
	fmt.Println("|---|---|---|")
	for _, k := range keys {
		c := tally[k]
		fmt.Printf("| %s | %d | %d |\n", k, c.missing, c.empty)
	}
	return nil
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
	fmt.Println("| File | Placement | Missing Props | Extra Props | Empty Props | Order |")
	fmt.Println("|---|---|---|---|---|---|")

	total, noFM := 0, 0
	missingPropsCount, extraPropsCount, emptyPropsCount, outOfOrderCount := 0, 0, 0, 0
	for _, file := range files {
		analysis, err := frontmatter.AnalyzeFile(file, template, templateKeys)
		if err != nil {
			fmt.Printf("| %s | error | | | | | %s |\n", displayPath(file, fmc.PathKeep), err)
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

		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			displayPath(file, fmc.PathKeep),
			analysis.Placement.Status,
			joinOrDash(analysis.MissingProps),
			joinOrDash(analysis.ExtraProps),
			joinOrDash(analysis.EmptyProps),
			order,
		)
	}

	fmt.Printf("\nFiles analyzed: %d\n", total)
	fmt.Println()
	fmt.Println("| Analysis Item | File Count |")
	fmt.Println("|---|---|")
	fmt.Printf("| Missing front matter | %d |\n", noFM)
	fmt.Printf("| Missing properties from template | %d |\n", missingPropsCount)
	fmt.Printf("| Extra properties | %d |\n", extraPropsCount)
	fmt.Printf("| Properties with empty values | %d |\n", emptyPropsCount)
	fmt.Printf("| Properties not in template order | %d |\n", outOfOrderCount)
	return nil
}

func (fmc *FrontMatterChecker) auditPlacement(files []string) error {
	fmt.Println("| FullPath | Placement | Reason | Candidate Start Line |")
	fmt.Println("|---|---|---|---|")

	results, err := frontmatter.AuditPlacementFiles(files)
	if err != nil {
		return err
	}

	for _, result := range results {
		startLine := ""
		if result.Candidate != nil {
			startLine = fmt.Sprintf("%d", result.Candidate.StartLine)
		}

		fmt.Printf("| %s | %s | %s | %s |\n",
			result.FilePath,
			result.Status,
			result.Reason,
			startLine,
		)
	}

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

		policy := frontmatter.PropertyPolicy{
			Key:    key,
			Action: action,
			Source: frontmatter.ValueSource(source),
		}
		switch frontmatter.ValueSource(source) {
		case frontmatter.SourceStatic:
			policy.StaticValue = value
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
		fmt.Println("| File | Present | Max Depth | Sub-properties |")
		fmt.Println("|---|---|---|---|")

		type nodeStats struct {
			depths    map[int]bool
			fileCount int
		}
		summary := map[string]*nodeStats{}

		for _, file := range files {
			label := displayPath(file, fmc.PathKeep)
			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("| %s | error | - | %v |\n", label, err)
				continue
			}
			insp, err := frontmatter.InspectProperty(string(content), propKey)
			if err != nil {
				fmt.Printf("| %s | error | - | %v |\n", label, err)
				continue
			}
			if !insp.Present {
				fmt.Printf("| %s | no | - | - |\n", label)
				continue
			}
			subKeys := make([]string, 0, len(insp.Nodes))
			seen := map[string]bool{}
			for _, n := range insp.Nodes {
				if !seen[n.Key] {
					subKeys = append(subKeys, n.Key)
					seen[n.Key] = true
				}
				if _, ok := summary[n.Key]; !ok {
					summary[n.Key] = &nodeStats{depths: map[int]bool{}}
				}
				summary[n.Key].depths[n.Depth] = true
			}
			for _, k := range subKeys {
				summary[k].fileCount++
			}
			depthStr := "-"
			if !insp.IsScalar {
				depthStr = fmt.Sprintf("%d", insp.MaxDepth)
			}
			subStr := "-"
			if len(subKeys) > 0 {
				subStr = strings.Join(subKeys, ", ")
			}
			fmt.Printf("| %s | yes | %s | %s |\n", label, depthStr, subStr)
		}

		if len(summary) > 0 {
			fmt.Println()
			fmt.Printf("### Summary\n\n")
			fmt.Println("| Sub-property | Depths | File Count |")
			fmt.Println("|---|---|---|")
			keys := make([]string, 0, len(summary))
			for k := range summary {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				st := summary[k]
				depths := make([]int, 0, len(st.depths))
				for d := range st.depths {
					depths = append(depths, d)
				}
				sort.Ints(depths)
				depthStrs := make([]string, 0, len(depths))
				for _, d := range depths {
					depthStrs = append(depthStrs, fmt.Sprintf("%d", d))
				}
				fmt.Printf("| %s | %s | %d |\n", k, strings.Join(depthStrs, ", "), st.fileCount)
			}
		}
		fmt.Println()
	}
	return nil
}

func (fmc *FrontMatterChecker) listEmpty(files []string) error {
	keys := []string(fmc.ListEmpty)

	fmt.Println("| File | Empty Props |")
	fmt.Println("|---|---|")

	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("| %s | error: %v |\n", displayPath(file, fmc.PathKeep), err)
			continue
		}
		empty, err := frontmatter.FindEmptyProps(string(content), keys)
		if err != nil {
			continue // no front matter or parse error — skip silently
		}
		if len(empty) == 0 {
			continue
		}
		fmt.Printf("| %s | %s |\n", displayPath(file, fmc.PathKeep), strings.Join(empty, ", "))
		for _, k := range empty {
			counts[k]++
		}
	}

	if len(counts) == 0 {
		fmt.Println("\nNo empty properties found.")
		return nil
	}

	type kv struct {
		Key   string
		Count int
	}
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

	fmt.Println("\nSummary:")
	fmt.Println("| Property | Count |")
	fmt.Println("|---|---|")
	for _, entry := range ranked {
		fmt.Printf("| %s | %d |\n", entry.Key, entry.Count)
	}
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
		plan, err := frontmatter.PlanRemoveIfEmpty(file, string(content), []string(fmc.RemoveEmpty))
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
	fmt.Println("| File | Extra Props |")
	fmt.Println("|---|---|")

	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("| %s | error: %v |\n", file, err)
			continue
		}
		extras, err := frontmatter.FindExtraProps(string(content), template)
		if err != nil {
			fmt.Printf("| %s | error: %v |\n", file, err)
			continue
		}
		fmt.Printf("| %s | %s |\n", file, joinOrDash(extras))
		for _, k := range extras {
			counts[k]++
		}
	}

	if len(counts) == 0 {
		fmt.Println("\nNo extra properties found.")
		return nil
	}

	type kv struct {
		Key   string
		Count int
	}
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

	fmt.Println("\nSummary:")
	fmt.Println("| Property | Count |")
	fmt.Println("|---|---|")
	for _, entry := range ranked {
		fmt.Printf("| %s | %d |\n", entry.Key, entry.Count)
	}

	return nil
}

func (fmc *FrontMatterChecker) listMissingProps(files []string, template map[string]any) error {
	fmt.Println("| File | Missing Props |")
	fmt.Println("|---|---|")

	counts := map[string]int{}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("| %s | error: %v |\n", file, err)
			continue
		}
		missing, err := frontmatter.FindMissingProps(string(content), template)
		if err != nil {
			fmt.Printf("| %s | error: %v |\n", file, err)
			continue
		}
		fmt.Printf("| %s | %s |\n", file, joinOrDash(missing))
		for _, k := range missing {
			counts[k]++
		}
	}

	if len(counts) == 0 {
		fmt.Println("\nNo missing properties found.")
		return nil
	}

	type kv struct {
		Key   string
		Count int
	}
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

	fmt.Println("\nSummary:")
	fmt.Println("| Property | Count |")
	fmt.Println("|---|---|")
	for _, entry := range ranked {
		fmt.Printf("| %s | %d |\n", entry.Key, entry.Count)
	}

	return nil
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

func (fmc *FrontMatterChecker) createSlugs(files []string) error {
	template := map[string]any{}
	policies := make([]frontmatter.PropertyPolicy, 0, len(fmc.CreateSlugs))

	for _, entry := range fmc.CreateSlugs {
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid -createSlug value %q: expected FromKey:ToKey[:action]", entry)
		}
		fromKey, toKey := parts[0], parts[1]

		action := frontmatter.ActionAddIfMissing
		if len(parts) == 3 {
			switch parts[2] {
			case "always":
				action = frontmatter.ActionOverwriteAlways
			case "if_empty":
				action = frontmatter.ActionOverwriteIfEmpty
			default:
				return fmt.Errorf("invalid action %q in -createSlug %q: expected always|if_empty", parts[2], entry)
			}
		}

		template[toKey] = ""
		policies = append(policies, frontmatter.PropertyPolicy{
			Key:     toKey,
			Action:  action,
			Source:  frontmatter.SourceTransform,
			Fn:      "slug",
			FromKey: fromKey,
		})
	}

	return fmc.fixFiles(files, template, policies)
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
