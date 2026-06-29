package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Field represents a parsed field from the schema
type Field struct {
	Path        string
	Type        string
	Required    bool
	Default     string
	Description string
	Enum        string
}

// RGD represents the parsed structure of the ResourceGraphDefinition YAML file
type RGD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Schema struct {
			APIVersion string    `yaml:"apiVersion"`
			Kind       string    `yaml:"kind"`
			Group      string    `yaml:"group"`
			Scope      string    `yaml:"scope"`
			Spec       yaml.Node `yaml:"spec"`
		} `yaml:"schema"`
	} `yaml:"spec"`
}

func main() {
	capabilitiesDir := "capabilities"
	outputDir := filepath.Join("docs", "gen")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", outputDir, err)
		os.Exit(1)
	}

	fmt.Printf("Searching for RGD files in %s...\n", capabilitiesDir)

	err := filepath.WalkDir(capabilitiesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Process only files ending with -rgd.yaml
		if !d.IsDir() && strings.HasSuffix(d.Name(), "-rgd.yaml") {
			fmt.Printf("Processing %s...\n", path)
			if err := generateDocForRGD(path, outputDir); err != nil {
				return fmt.Errorf("processing %s: %w", path, err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Documentation generation completed successfully!")
}

// generateDocForRGD parses a single RGD file and writes the generated markdown documentation
func generateDocForRGD(rgdPath, outputDir string) error {
	data, err := os.ReadFile(rgdPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var rgd RGD
	if err := yaml.Unmarshal(data, &rgd); err != nil {
		return fmt.Errorf("unmarshalling yaml: %w", err)
	}

	var fields []Field
	traverseNode("", &rgd.Spec.Schema.Spec, &fields)

	// Determine output filename: docs/gen/<kind_lowercase>-<version>.md
	outFilename := fmt.Sprintf("%s-%s.md", strings.ToLower(rgd.Spec.Schema.Kind), strings.ToLower(rgd.Spec.Schema.APIVersion))
	outPath := filepath.Join(outputDir, outFilename)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s API Schema Reference\n\n", rgd.Spec.Schema.Kind))
	sb.WriteString(fmt.Sprintf("- **Group:** `%s`\n", rgd.Spec.Schema.Group))
	sb.WriteString(fmt.Sprintf("- **Version:** `%s`\n", rgd.Spec.Schema.APIVersion))
	sb.WriteString(fmt.Sprintf("- **Scope:** `%s`\n\n", rgd.Spec.Schema.Scope))
	sb.WriteString("## Spec Schema\n\n")
	sb.WriteString("| Field | Type | Required | Default | Description |\n")
	sb.WriteString("|---|---|---|---|---|\n")

	for _, field := range fields {
		reqStr := "No"
		if field.Required {
			reqStr = "Yes"
		}

		defStr := "-"
		if field.Default != "" {
			defStr = fmt.Sprintf("`%s`", field.Default)
		}

		descStr := field.Description
		if field.Enum != "" {
			enumDesc := fmt.Sprintf(" (Enum: `%s`)", strings.ReplaceAll(field.Enum, ",", "`, `"))
			if descStr != "" {
				descStr += enumDesc
			} else {
				descStr = "Allowed values" + enumDesc
			}
		}
		if descStr == "" {
			descStr = "-"
		}

		sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s |\n",
			field.Path, field.Type, reqStr, defStr, descStr))
	}

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	fmt.Printf("Generated: %s\n", outPath)
	return nil
}

// traverseNode recursively walks the yaml.Node mapping schema to find all fields in order
func traverseNode(prefix string, node *yaml.Node, fields *[]Field) {
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			key := keyNode.Value
			fullPath := key
			if prefix != "" {
				fullPath = prefix + "." + key
			}

			switch valNode.Kind {
			case yaml.ScalarNode:
				fieldType, required, defaultValue, description, enum := parseFieldLine(valNode.Value)
				*fields = append(*fields, Field{
					Path:        fullPath,
					Type:        fieldType,
					Required:    required,
					Default:     defaultValue,
					Description: description,
					Enum:        enum,
				})
			case yaml.MappingNode:
				traverseNode(fullPath, valNode, fields)
			}
		}
	}
}

// parseFieldLine extracts type, required flag, default value, enum, and description from a schema field line
func parseFieldLine(line string) (fieldType string, required bool, defaultValue string, description string, enum string) {
	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, "|", 2)
	fieldType = strings.TrimSpace(parts[0])

	// If there's a quoted type definition, clean the quotes (e.g. "[]string | default=[]" or "map[string]string")
	if strings.HasPrefix(fieldType, "\"") && strings.HasSuffix(fieldType, "\"") && len(fieldType) >= 2 {
		fieldType = fieldType[1 : len(fieldType)-1]
	}

	if len(parts) < 2 {
		return
	}
	markersStr := strings.TrimSpace(parts[1])

	// Handle case where entire line was quoted, e.g. "[]string | default=[]"
	if strings.HasSuffix(markersStr, "\"") && !strings.Contains(markersStr, "\\\"") {
		markersStr = markersStr[:len(markersStr)-1]
	}

	i := 0
	n := len(markersStr)
	for i < n {
		// skip whitespace
		for i < n && unicode.IsSpace(rune(markersStr[i])) {
			i++
		}
		if i >= n {
			break
		}

		// read key
		startKey := i
		for i < n && markersStr[i] != '=' && !unicode.IsSpace(rune(markersStr[i])) {
			i++
		}
		key := markersStr[startKey:i]

		if i < n && markersStr[i] == '=' {
			i++ // skip '='
			var val string
			if i < n && markersStr[i] == '"' {
				i++ // skip opening quote
				startVal := i
				for i < n && markersStr[i] != '"' {
					// handle escaping if needed, but not present in current RGDs
					i++
				}
				val = markersStr[startVal:i]
				if i < n {
					i++ // skip closing quote
				}
			} else {
				startVal := i
				for i < n && !unicode.IsSpace(rune(markersStr[i])) {
					i++
				}
				val = markersStr[startVal:i]
			}

			switch key {
			case "required":
				required = (val == "true")
			case "default":
				defaultValue = val
			case "description":
				description = val
			case "enum":
				enum = val
			}
		}
	}
	return
}
