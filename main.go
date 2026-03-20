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

	IssuesOnly     bool
	Verbose        bool
	ListExtraProps bool
}

func main() {
	// Subcommands are checked before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "policy" {
		runPolicyCommand(os.Args[2:])
		return
	}

	checker := &FrontMatterChecker{
		FixOptions: make(map[string]bool),
	}

	// Parse flags
	flag.StringVar(&checker.TemplateFile, "template", "", "Path to the front matter template file")
	flag.StringVar(&checker.TemplateFile, "t", "", "Alias for -template")
	flag.StringVar(&checker.Dir, "dir", "", "Directory containing markdown files")
	flag.StringVar(&checker.ConfigFile, "config", "", "Path to the configuration JSON file")
	flag.StringVar(&checker.PolicyFile, "policy", "", "Path to the property policy JSON file")
	flag.StringVar(&checker.PolicyFile, "p", "", "Alias for -policy")
	files := flag.String("files", "", "Comma-separated list of files to analyze/fix")
	issuesOnly := flag.Bool("issues-only", false, "Show only files with issues")
	verbose := flag.Bool("verbose", false, "Show more detailed analysis output")
	placementAudit := flag.Bool("placementAudit", false, "Audit front matter placement only")
	analyzeOnly := flag.Bool("analyze", false, "Analyze the files without making changes")
	fixFullConform := flag.Bool("fullConform", false, "Fully conform the front matter to the template")
	fixAllProps := flag.Bool("allProps", false, "Ensure all properties in the template exist in the front matter")
	fixOrder := flag.Bool("fixOrder", false, "Reorder properties to match the template")
	listExtraProps := flag.Bool("listExtraProps", false, "List properties not defined in the template")
	removeExtraProps := flag.Bool("removeExtraProps", false, "Remove properties not defined in the template")
	genID := flag.Bool("genID", false, "Generate IDs for files where the ID property is missing or empty")
	help := flag.Bool("help", false, "Display help information")
	flag.Parse()

	if *help {
		fmt.Println("Usage: [flags]")
		flag.PrintDefaults()
		return
	}

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
	checker.FixOptions["removeExtraProps"] = *removeExtraProps
	checker.GenID = *genID
	checker.ListExtraProps = *listExtraProps

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

	template, err := fmc.loadTemplate()
	if err != nil {
		return err
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

	if fmc.AnalyzeOnly {
		return fmc.analyzeFiles(filesToProcess, template)
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

func (fmc *FrontMatterChecker) analyzeFiles(files []string, template map[string]any) error {
	fmt.Println("| FullPath | Placement | Missing Props | Extra Props | Reason |")
	fmt.Println("|---|---|---|---|---|")

	for _, file := range files {
		analysis, err := frontmatter.AnalyzeFile(file, template)
		if err != nil {
			fmt.Printf("| %s | error |  |  | %s |\n", file, err)
			continue
		}

		if fmc.IssuesOnly && !analysis.HasIssues() {
			continue
		}

		fmt.Printf(
			"| %s | %s | %s | %s | %s |\n",
			analysis.Path,
			analysis.Placement.Status,
			joinOrDash(analysis.MissingProps),
			joinOrDash(analysis.ExtraProps),
			valueOrDash(analysis.Placement.Reason),
		)

		if fmc.Verbose {
			// print more detail lines later
		}
	}

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
