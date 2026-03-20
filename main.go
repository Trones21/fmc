package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Trones21/fmc/frontmatter"
)

type Config struct {
	ValueInsertion map[string]interface{} `json:"valueInsertion"`
}

type FrontMatterChecker struct {
	TemplateFile       string
	Dir                string
	Files              []string
	ConfigFile         string
	FixOptions         map[string]bool
	AnalyzeOnly        bool
	PlacementAuditOnly bool
	GenID              bool
	Config             Config

	IssuesOnly bool
	Verbose    bool
}

func main() {
	checker := &FrontMatterChecker{
		FixOptions: make(map[string]bool),
	}

	// Parse flags
	flag.StringVar(&checker.TemplateFile, "template", "", "Path to the front matter template file")
	flag.StringVar(&checker.TemplateFile, "t", "", "Alias for -template")
	flag.StringVar(&checker.Dir, "dir", "", "Directory containing markdown files")
	flag.StringVar(&checker.ConfigFile, "config", "", "Path to the configuration JSON file")
	files := flag.String("files", "", "Comma-separated list of files to analyze/fix")
	issuesOnly := flag.Bool("issues-only", false, "Show only files with issues")
	verbose := flag.Bool("verbose", false, "Show more detailed analysis output")
	placementAudit := flag.Bool("placementAudit", false, "Audit front matter placement only")
	analyzeOnly := flag.Bool("analyze", false, "Analyze the files without making changes")
	fixFullConform := flag.Bool("fullConform", false, "Fully conform the front matter to the template")
	fixAllProps := flag.Bool("allProps", false, "Ensure all properties in the template exist in the front matter")
	fixOrder := flag.Bool("fixOrder", false, "Reorder properties to match the template")
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

	if fmc.AnalyzeOnly {
		return fmc.analyzeFiles(filesToProcess, template)
	}

	return fmc.fixFiles(filesToProcess, template)
}

func (fmc *FrontMatterChecker) loadTemplate() (map[string]interface{}, error) {
	file, err := os.Open(fmc.TemplateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open template file: %v", err)
	}
	defer file.Close()

	template := make(map[string]interface{})
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

func (fmc *FrontMatterChecker) analyzeFiles(files []string, template map[string]interface{}) error {
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

func (fmc *FrontMatterChecker) fixFiles(files []string, template map[string]interface{}) error {
	for _, file := range files {
		fmt.Printf("Fixing file: %s (implementation to be added)\n", file)
	}

	return nil
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
