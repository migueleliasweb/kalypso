package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

type WorkloadStats struct {
	Name         string
	InputLines   int
	InputBytes   int
	OutputLines  int
	OutputBytes  int
	SavedLinesPct float64
	SavedBytesPct float64
}

func main() {
	workloads := []string{
		"core-minimal",
		"core-full",
		"core-statefulset",
		"core-daemonset",
		"core-rbac",
		"core-networkpolicy",
	}

	var stats []WorkloadStats

	for _, name := range workloads {
		stat, err := processWorkload(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing workload %s: %v\n", name, err)
			continue
		}
		stats = append(stats, stat)
	}

	printMarkdownTable(stats)
}

func processWorkload(name string) (WorkloadStats, error) {
	// 1. Read input file
	inputPath := filepath.Join("e2e", "testdata", name+".yaml")
	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return WorkloadStats{}, fmt.Errorf("reading input file: %w", err)
	}

	// Clean input bytes of comments and trailing newlines for fair line count
	inputClean := cleanManifestText(string(inputBytes))
	inputLines := countLines(inputClean)
	inputByteLen := len(inputClean)

	// 2. Query cluster for generated resources
	kinds := "deployment,statefulset,daemonset,sa,pdb,hpa,cm,secret,role,rolebinding,clusterrole,clusterrolebinding,networkpolicy"
	cmd := exec.Command("kubectl", "--context", "kind-kalypso-e2e", "get", kinds,
		"-l", fmt.Sprintf("kro.run/instance-name=%s", name), "-A", "-o", "json")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return WorkloadStats{}, fmt.Errorf("running kubectl get: %w, stderr: %s", err, stderr.String())
	}

	// 3. Parse JSON list of resources
	var list struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return WorkloadStats{}, fmt.Errorf("unmarshalling kubectl JSON: %w", err)
	}

	// 4. Clean each resource and marshal to YAML
	var outputYAMLs []string
	for _, item := range list.Items {
		cleaned := cleanResource(item)
		yamlBytes, err := yaml.Marshal(cleaned)
		if err != nil {
			return WorkloadStats{}, fmt.Errorf("marshalling cleaned resource: %w", err)
		}
		outputYAMLs = append(outputYAMLs, string(yamlBytes))
	}

	// 5. Combine outputs
	combinedOutput := strings.Join(outputYAMLs, "---\n")
	combinedOutput = cleanManifestText(combinedOutput)

	outputLines := countLines(combinedOutput)
	outputByteLen := len(combinedOutput)

	savedLinesPct := 0.0
	if outputLines > 0 {
		savedLinesPct = float64(outputLines-inputLines) / float64(outputLines) * 100.0
	}

	savedBytesPct := 0.0
	if outputByteLen > 0 {
		savedBytesPct = float64(outputByteLen-inputByteLen) / float64(outputByteLen) * 100.0
	}

	return WorkloadStats{
		Name:         name,
		InputLines:   inputLines,
		InputBytes:   inputByteLen,
		OutputLines:  outputLines,
		OutputBytes:  outputByteLen,
		SavedLinesPct: savedLinesPct,
		SavedBytesPct: savedBytesPct,
	}, nil
}

func cleanResource(item map[string]interface{}) map[string]interface{} {
	// Remove status
	delete(item, "status")

	// Clean metadata
	if metadata, ok := item["metadata"].(map[string]interface{}); ok {
		delete(metadata, "uid")
		delete(metadata, "resourceVersion")
		delete(metadata, "generation")
		delete(metadata, "creationTimestamp")
		delete(metadata, "ownerReferences")
		delete(metadata, "managedFields")

		// Remove unwanted annotations
		if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
			for k := range annotations {
				if strings.HasPrefix(k, "kro.run/") || strings.HasPrefix(k, "applyset.kubernetes.io/") || k == "deployment.kubernetes.io/revision" {
					delete(annotations, k)
				}
			}
			if len(annotations) == 0 {
				delete(metadata, "annotations")
			}
		}

		// Remove unwanted labels
		if labels, ok := metadata["labels"].(map[string]interface{}); ok {
			for k := range labels {
				if strings.HasPrefix(k, "kro.run/") || strings.HasPrefix(k, "applyset.kubernetes.io/") || k == "app.kubernetes.io/managed-by" {
					delete(labels, k)
				}
			}
			if len(labels) == 0 {
				delete(metadata, "labels")
			}
		}
	}

	return item
}

func cleanManifestText(text string) string {
	// Split by newline and filter out comments and empty lines
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comment-only lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

func countLines(text string) int {
	if len(strings.TrimSpace(text)) == 0 {
		return 0
	}
	return len(strings.Split(text, "\n"))
}

func printMarkdownTable(stats []WorkloadStats) {
	var buf bytes.Buffer
	buf.WriteString("# Configuration Savings Analysis\n\n")
	buf.WriteString("Below is the comparison of configuration size between **Kalypso custom resources** (high-level specification) and the **raw Kubernetes resources** automatically generated by KRO in the cluster.\n\n")
	buf.WriteString("| Workload Case | Kalypso Lines (LOC) | Raw K8s Lines (LOC) | Savings (Lines %) | Kalypso Bytes | Raw K8s Bytes | Savings (Bytes %) |\n")
	buf.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")

	for _, s := range stats {
		buf.WriteString(fmt.Sprintf("| `%s` | %d | %d | **%.1f%%** | %d B | %d B | **%.1f%%** |\n",
			s.Name, s.InputLines, s.OutputLines, s.SavedLinesPct, s.InputBytes, s.OutputBytes, s.SavedBytesPct))
	}

	fmt.Println(buf.String())
}
