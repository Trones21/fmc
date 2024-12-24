package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ValueInsertion map[string]interface{} `json:"valueInsertion"`
}

type FrontMatterChecker struct {
	TemplateFile string
	Dir          string
	Files        []string
	ConfigFile   string
	FixOptions   map[string]bool
	AnalyzeOnly  bool
	GenID        bool
	Config       Config
}

func main() {
	checker := &FrontMatterChecker{
		FixOptions: make(map[string]bool),
	}

	// Parse flags
	flag.StringVar(&checker.TemplateFile, "template", "", "Path to the front matter template file")
	flag.StringVar(&checker.Dir, "dir", "", "Directory containing markdown files")
	flag.StringVar(&checker.ConfigFile, "config", "", "Path to the configuration JSON file")
	files := flag.String("files", "", "Comma-separated list of files to analyze/fix")
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

	checker.AnalyzeOnly = *analyzeOnly
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
	// Load template
	template, err := fmc.loadTemplate()
	if err != nil {
		return err
	}

	// Load config if specified
	if fmc.ConfigFile != "" {
		if err := fmc.loadConfig(); err != nil {
			return err
		}
	}

	// Get files to process
	filesToProcess, err := fmc.getFiles()
	if err != nil {
		return err
	}

	// Analyze or fix files
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
	fmt.Println("| FullPath | Template Properties | Extra Properties |")
	fmt.Println("|---|---|---|")

	for _, file := range files {
		fmt.Printf("| %s | analysis to be implemented |\n", file)
	}

	return nil
}

func (fmc *FrontMatterChecker) fixFiles(files []string, template map[string]interface{}) error {
	for _, file := range files {
		fmt.Printf("Fixing file: %s (implementation to be added)\n", file)
	}

	return nil
}
