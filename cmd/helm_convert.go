package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	sigyaml "sigs.k8s.io/yaml"
)

var (
	outputFile  string
	releaseName string
	targetNS    string
)

var helmConvertCmd = &cobra.Command{
	Use:   "helm-convert <chart-path> [flags] [-- [helm-template-flags]]",
	Short: "Convert a Helm chart into a Kalypso Compute CR",
	Long: `helm-convert renders a Helm chart using 'helm template' and processes the resulting resources
(Deployment/StatefulSet/DaemonSet, ServiceAccount, ConfigMap, Secret, HPA, PDB)
into a single Kalypso Compute custom resource.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]

		// Check if helm is available locally
		if _, err := exec.LookPath("helm"); err != nil {
			fmt.Fprintln(os.Stderr, "error: helm is not available locally. Please install helm.")
			os.Exit(1)
		}

		// Prepare helm template arguments
		helmArgs := []string{"template", releaseName, chartPath}
		if targetNS != "" {
			helmArgs = append(helmArgs, "--namespace", targetNS)
		}

		// Check if there are arguments after "--" to forward to helm template
		// Cobra automatically handles "--" by separating them, but we can also parse the raw OS args
		dashDashIndex := -1
		for i, arg := range os.Args {
			if arg == "--" {
				dashDashIndex = i
				break
			}
		}

		if dashDashIndex != -1 && dashDashIndex+1 < len(os.Args) {
			helmArgs = append(helmArgs, os.Args[dashDashIndex+1:]...)
		}

		// Run helm template command
		helmCmd := exec.Command("helm", helmArgs...)
		var stdout, stderr bytes.Buffer
		helmCmd.Stdout = &stdout
		helmCmd.Stderr = &stderr

		if err := helmCmd.Run(); err != nil {
			return fmt.Errorf("helm template failed: %w\nstderr: %s", err, stderr.String())
		}

		// Convert manifests
		computeYAML, hasSecrets, err := ConvertManifests(stdout.Bytes(), targetNS)
		if err != nil {
			return fmt.Errorf("failed to convert manifests: %w", err)
		}

		// Output result
		if outputFile != "" {
			err = os.WriteFile(outputFile, computeYAML, 0o644)
			if err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
		} else {
			fmt.Print(string(computeYAML))
		}

		// Warning on secrets
		if hasSecrets {
			fmt.Fprintln(os.Stderr, "\nWARNING: Mapping secrets into a Kalypso custom resource is not recommended for production environments. Be careful not to commit or version control files containing raw secret data.")
		}

		return nil
	},
}

func init() {
	helmConvertCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default is stdout)")
	helmConvertCmd.Flags().StringVar(&releaseName, "release-name", "kalypso-release", "Release name for helm template")
	helmConvertCmd.Flags().StringVarP(&targetNS, "namespace", "n", "", "Target namespace for the Kalypso CR")
	rootCmd.AddCommand(helmConvertCmd)
}

// ConvertManifests parses multi-document YAML and constructs a Compute CR
func ConvertManifests(data []byte, targetNamespace string) ([]byte, bool, error) {
	var resources []*unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)

	for {
		var obj unstructured.Unstructured
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, false, err
		}

		if obj.GetKind() == "" {
			continue
		}

		resources = append(resources, &obj)
	}

	var workloads []*unstructured.Unstructured
	var hpas []*unstructured.Unstructured
	var pdbs []*unstructured.Unstructured
	var serviceAccounts []*unstructured.Unstructured
	var configMaps []*unstructured.Unstructured
	var secrets []*unstructured.Unstructured

	for _, r := range resources {
		switch r.GetKind() {
		case "Deployment", "StatefulSet", "DaemonSet":
			workloads = append(workloads, r)
		case "HorizontalPodAutoscaler":
			hpas = append(hpas, r)
		case "PodDisruptionBudget":
			pdbs = append(pdbs, r)
		case "ServiceAccount":
			serviceAccounts = append(serviceAccounts, r)
		case "ConfigMap":
			configMaps = append(configMaps, r)
		case "Secret":
			secrets = append(secrets, r)
		}
	}

	if len(workloads) == 0 {
		return nil, false, fmt.Errorf("no Deployment, StatefulSet, or DaemonSet found in Helm chart template output")
	}

	if len(workloads) > 1 {
		fmt.Fprintf(os.Stderr, "warning: multiple workloads found in Helm chart, using the first one: %s\n", workloads[0].GetName())
	}

	workload := workloads[0]
	workloadType := workload.GetKind()
	workloadName := workload.GetName()

	namespace := targetNamespace
	if namespace == "" {
		namespace = workload.GetNamespace()
	}
	if namespace == "" {
		namespace = "default"
	}

	spec := map[string]interface{}{
		"workloadType": workloadType,
	}

	// Replicas
	if workloadType == "Deployment" || workloadType == "StatefulSet" {
		replicas, found, _ := unstructured.NestedInt64(workload.Object, "spec", "replicas")
		if found {
			spec["replicas"] = replicas
		}
	}

	// Pod Spec extraction
	containers, found, _ := unstructured.NestedSlice(workload.Object, "spec", "template", "spec", "containers")
	if !found || len(containers) == 0 {
		return nil, false, fmt.Errorf("workload %s has no containers", workloadName)
	}

	// Choose main container: matching workload name, or "app"/"main", or first
	var container map[string]interface{}
	for _, c := range containers {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		cName, _ := cMap["name"].(string)
		if cName == workloadName || cName == "app" || cName == "main" {
			container = cMap
			break
		}
	}
	if container == nil {
		container = containers[0].(map[string]interface{})
	}

	// Image
	if img, ok := container["image"].(string); ok {
		spec["image"] = img
	}

	// Ports
	var containerPorts []interface{}
	if pts, ok := container["ports"].([]interface{}); ok {
		containerPorts = pts
		if len(pts) > 0 {
			if pMap, ok := pts[0].(map[string]interface{}); ok {
				if cp, ok := pMap["containerPort"]; ok {
					if cpInt, ok := cp.(float64); ok {
						spec["port"] = int64(cpInt)
					} else if cpInt, ok := cp.(int64); ok {
						spec["port"] = cpInt
					}
				}
			}
		}
	}

	// Command & Args & Env
	if cmdSlice, ok := container["command"].([]interface{}); ok {
		spec["command"] = cmdSlice
	}
	if argsSlice, ok := container["args"].([]interface{}); ok {
		spec["args"] = argsSlice
	}
	if envSlice, ok := container["env"].([]interface{}); ok {
		spec["env"] = envSlice
	}

	// Resources (requests and limits)
	resMap, found, _ := unstructured.NestedMap(container, "resources")
	if found {
		requests, _, _ := unstructured.NestedMap(resMap, "requests")
		limits, _, _ := unstructured.NestedMap(resMap, "limits")

		specRes := map[string]interface{}{}

		if len(requests) > 0 {
			reqMap := map[string]interface{}{}
			if cpu, ok := resourceValToString(requests["cpu"]); ok {
				reqMap["cpu"] = cpu
			}
			if mem, ok := resourceValToString(requests["memory"]); ok {
				reqMap["memory"] = mem
			}
			if len(reqMap) > 0 {
				specRes["requests"] = reqMap
			}
		}

		if len(limits) > 0 {
			limMap := map[string]interface{}{}
			if cpu, ok := resourceValToString(limits["cpu"]); ok {
				limMap["cpu"] = cpu
			}
			if mem, ok := resourceValToString(limits["memory"]); ok {
				limMap["memory"] = mem
			}
			if len(limMap) > 0 {
				specRes["limits"] = limMap
			}
		}

		if len(specRes) > 0 {
			spec["resources"] = specRes
		}
	}

	// Probes
	probes := map[string]interface{}{}
	probeNames := []struct {
		k8sKey   string
		rgdKey   string
		defaults map[string]interface{}
	}{
		{"livenessProbe", "liveness", map[string]interface{}{"enabled": true, "path": "/healthz", "port": int64(8080)}},
		{"readinessProbe", "readiness", map[string]interface{}{"enabled": true, "path": "/readyz", "port": int64(8080)}},
		{"startupProbe", "startup", map[string]interface{}{"enabled": false, "path": "/healthz", "port": int64(8080)}},
	}

	for _, p := range probeNames {
		pMap, found, _ := unstructured.NestedMap(container, p.k8sKey)
		if found {
			rProbe := map[string]interface{}{"enabled": true}
			httpGet, httpFound, _ := unstructured.NestedMap(pMap, "httpGet")
			if httpFound {
				if path, ok := httpGet["path"].(string); ok {
					rProbe["path"] = path
				}
				if port, ok := httpGet["port"]; ok {
					resolved := resolvePort(port, containerPorts)
					rProbe["port"] = resolved
				}
				for _, field := range []string{"initialDelaySeconds", "periodSeconds", "timeoutSeconds", "successThreshold", "failureThreshold"} {
					if val, ok := pMap[field]; ok {
						if valInt, ok := val.(float64); ok {
							rProbe[field] = int64(valInt)
						} else if valInt, ok := val.(int64); ok {
							rProbe[field] = valInt
						}
					}
				}
			} else {
				// If not httpGet, map as custom probe config
				rProbe["custom"] = pMap
			}
			probes[p.rgdKey] = rProbe
		} else {
			// Explicitly set enabled: false if not configured in workload to override defaults
			probes[p.rgdKey] = map[string]interface{}{"enabled": false}
		}
	}
	spec["probes"] = probes

	// Restart Policy
	if rp, ok := container["restartPolicy"].(string); ok {
		spec["restartPolicy"] = rp
	}

	// Volumes & VolumeMounts
	vols, foundVols, _ := unstructured.NestedSlice(workload.Object, "spec", "template", "spec", "volumes")
	if foundVols && len(vols) > 0 {
		spec["volumes"] = vols
	}
	vMnts, foundVMnts, _ := unstructured.NestedSlice(container, "volumeMounts")
	if foundVMnts && len(vMnts) > 0 {
		spec["volumeMounts"] = vMnts
	}

	// Service Account
	saConfig := map[string]interface{}{}
	if len(serviceAccounts) > 0 {
		saConfig["create"] = true
		saConfig["name"] = serviceAccounts[0].GetName()
	} else {
		saName, _, _ := unstructured.NestedString(workload.Object, "spec", "template", "spec", "serviceAccountName")
		if saName != "" && saName != "default" {
			saConfig["create"] = false
			saConfig["name"] = saName
		}
	}
	if len(saConfig) > 0 {
		spec["serviceAccount"] = saConfig
	}

	// Scheduling & Affinity
	scheduling := map[string]interface{}{}
	nodeSelector, _, _ := unstructured.NestedMap(workload.Object, "spec", "template", "spec", "nodeSelector")
	if len(nodeSelector) > 0 {
		scheduling["nodeSelector"] = nodeSelector
	}
	affinity, _, _ := unstructured.NestedMap(workload.Object, "spec", "template", "spec", "affinity")
	if len(affinity) > 0 {
		scheduling["affinity"] = affinity
	}
	tolerations, _, _ := unstructured.NestedSlice(workload.Object, "spec", "template", "spec", "tolerations")
	if len(tolerations) > 0 {
		scheduling["tolerations"] = tolerations
	}

	// Topology Spread Constraints
	tsc, _, _ := unstructured.NestedSlice(workload.Object, "spec", "template", "spec", "topologySpreadConstraints")
	if len(tsc) > 0 {
		ts := map[string]interface{}{
			"enabled": true,
		}
		first := tsc[0].(map[string]interface{})
		if skew, ok := first["maxSkew"]; ok {
			if skewF, ok := skew.(float64); ok {
				ts["maxSkew"] = int64(skewF)
			} else if skewI, ok := skew.(int64); ok {
				ts["maxSkew"] = skewI
			}
		}
		if key, ok := first["topologyKey"].(string); ok {
			ts["topologyKey"] = key
		}
		if unsatisfiable, ok := first["whenUnsatisfiable"].(string); ok {
			ts["whenUnsatisfiable"] = unsatisfiable
		}
		var custom []interface{}
		if len(tsc) > 1 {
			custom = tsc[1:]
		} else {
			custom = []interface{}{}
		}
		ts["customConstraints"] = custom
		scheduling["topologySpread"] = ts
	}
	if len(scheduling) > 0 {
		spec["scheduling"] = scheduling
	}

	// ConfigMaps
	if len(configMaps) > 0 {
		cmData := map[string]string{}
		for _, cm := range configMaps {
			dataMap, found, _ := unstructured.NestedStringMap(cm.Object, "data")
			if found {
				for k, v := range dataMap {
					cmData[k] = v
				}
			}
		}
		spec["configMap"] = map[string]interface{}{
			"enabled": true,
			"data":    cmData,
		}
	}

	// Secrets
	hasSecrets := false
	if len(secrets) > 0 {
		hasSecrets = true
		secretData := map[string]string{}
		for _, sec := range secrets {
			// Extract standard encoded data
			dataMap, found, _ := unstructured.NestedStringMap(sec.Object, "data")
			if found {
				for k, v := range dataMap {
					decoded, err := base64.StdEncoding.DecodeString(v)
					if err == nil {
						secretData[k] = string(decoded)
					} else {
						secretData[k] = v
					}
				}
			}
			// Extract unencoded stringData if present
			stringDataMap, found, _ := unstructured.NestedStringMap(sec.Object, "stringData")
			if found {
				for k, v := range stringDataMap {
					secretData[k] = v
				}
			}
		}
		spec["secret"] = map[string]interface{}{
			"enabled": true,
			"data":    secretData,
		}
	}

	// PDB
	if len(pdbs) > 0 {
		pdb := pdbs[0]
		maxUnavail, found, _ := unstructured.NestedFieldNoCopy(pdb.Object, "spec", "maxUnavailable")
		if found {
			var valStr string
			switch v := maxUnavail.(type) {
			case string:
				valStr = v
			case int64:
				valStr = fmt.Sprintf("%d", v)
			case float64:
				valStr = fmt.Sprintf("%d", int64(v))
			default:
				valStr = fmt.Sprintf("%v", v)
			}
			spec["pdb"] = map[string]interface{}{
				"enabled":        true,
				"maxUnavailable": valStr,
			}
		}
	}

	// Autoscaling (HPA)
	if len(hpas) > 0 {
		hpa := hpas[0]
		minRep, minFound, _ := unstructured.NestedInt64(hpa.Object, "spec", "minReplicas")
		maxRep, maxFound, _ := unstructured.NestedInt64(hpa.Object, "spec", "maxReplicas")

		auto := map[string]interface{}{
			"enabled": true,
		}
		if minFound {
			auto["minReplicas"] = minRep
		} else {
			auto["minReplicas"] = int64(1)
		}
		if maxFound {
			auto["maxReplicas"] = maxRep
		}

		// Find target CPU metric
		metrics, metricsFound, _ := unstructured.NestedSlice(hpa.Object, "spec", "metrics")
		if metricsFound {
			for _, m := range metrics {
				mMap, ok := m.(map[string]interface{})
				if !ok {
					continue
				}
				mType, _ := mMap["type"].(string)
				if mType == "Resource" {
					res, resFound, _ := unstructured.NestedMap(mMap, "resource")
					if resFound && res["name"] == "cpu" {
						target, targetFound, _ := unstructured.NestedMap(res, "target")
						if targetFound && target["type"] == "Utilization" {
							if avg, ok := target["averageUtilization"]; ok {
								if avgF, ok := avg.(float64); ok {
									auto["targetCPUUtilization"] = int64(avgF)
								} else if avgI, ok := avg.(int64); ok {
									auto["targetCPUUtilization"] = avgI
								}
							}
						}
					}
				}
			}
		}
		spec["autoscaling"] = auto
	}

	// StatefulSet volumeClaimTemplates mapping
	if workloadType == "StatefulSet" {
		vct, vctFound, _ := unstructured.NestedSlice(workload.Object, "spec", "volumeClaimTemplates")
		if vctFound {
			spec["volumeClaimTemplates"] = vct
		}
	}

	// Final CR mapping
	compute := map[string]interface{}{
		"apiVersion": "kalypso.lmoet.io/v1alpha2",
		"kind":       "Compute",
		"metadata": map[string]interface{}{
			"name":      workloadName,
			"namespace": namespace,
		},
		"spec": spec,
	}

	// Copy standard labels/annotations from the workload if present
	if labels := workload.GetLabels(); len(labels) > 0 {
		// Clean standard helm charts labels that are release specific or metadata.labels
		cleanedLabels := map[string]string{}
		for k, v := range labels {
			// Avoid copying helm release or system keys if they interfere, but usually it's fine
			cleanedLabels[k] = v
		}
		if len(cleanedLabels) > 0 {
			compute["metadata"].(map[string]interface{})["labels"] = cleanedLabels
		}
	}

	yamlBytes, err := sigyaml.Marshal(compute)
	if err != nil {
		return nil, false, err
	}

	return yamlBytes, hasSecrets, nil
}

func resolvePort(portVal interface{}, containerPorts []interface{}) int64 {
	switch v := portVal.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		for _, p := range containerPorts {
			pMap, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := pMap["name"].(string)
			if name == v {
				if cp, ok := pMap["containerPort"]; ok {
					if cpInt, ok := cp.(float64); ok {
						return int64(cpInt)
					}
					if cpInt, ok := cp.(int64); ok {
						return cpInt
					}
					if cpInt, ok := cp.(int); ok {
						return int64(cpInt)
					}
				}
			}
		}
	}
	return 8080 // default fallback
}

func resourceValToString(val interface{}) (string, bool) {
	if val == nil {
		return "", false
	}
	switch v := val.(type) {
	case string:
		return v, true
	case int:
		return fmt.Sprintf("%d", v), true
	case int64:
		return fmt.Sprintf("%d", v), true
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v)), true
		}
		return fmt.Sprintf("%g", v), true
	}
	return fmt.Sprintf("%v", val), true
}

