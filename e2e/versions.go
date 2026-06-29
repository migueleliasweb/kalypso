package e2e

// manifestSpec describes a third-party manifest the e2e suite needs in the
// cluster. The suite downloads it on demand into crdManifestsDir (gitignored)
// and caches it across runs, so a fresh CI checkout fetches once and reuses.
type manifestSpec struct {
	url      string
	filename string
}

// Pinned versions — bump these (and the URLs resolve automatically) to upgrade.
const (
	kroVersion          = "0.9.2"
	istioVersion        = "1.29.2"
	promOperatorVersion = "0.92.0"

	// nodeImage must be compatible with the locally installed kind binary.
	nodeImage = "kindest/node:v1.32.2"
)

// Cluster + filesystem layout.
const (
	clusterName    = "kalypso-e2e"
	kindConfigFile = "kind-config.yaml"
	logsDir        = "./logs"

	// destroyEnvVar gates teardown. By default the cluster is left running for
	// troubleshooting; set this truthy to destroy it on finish.
	destroyEnvVar = "KRO_E2E_DESTROY_CLUSTER"

	// crdManifestsDir caches downloaded third-party manifests (gitignored).
	crdManifestsDir = "crd-manifests"
)

// KRO install bundle facts (derived from the v0.9.2 release manifest).
const (
	kroNamespace      = "kro-system"
	kroDeploymentName = "kro"
)

// The RGDs under test and the APIs they generate.
const (
	podSpecRGDPath = "../capabilities/podspec/v1alpha1/podspec-rgd.yaml"
	podSpecRGDName = "podspec"
	podSpecCRDName = "podspecs.kalypso.lmoet.io"

	rgdPath     = "../capabilities/core/v1alpha2/core-rgd.yaml"
	rgdName     = "core"
	coreCRDName = "cores.kalypso.lmoet.io"

	workloadGroup   = "kalypso.lmoet.io"
	workloadVersion = "v1alpha2"
	workloadKind    = "Core"

	// testNamespace is where namespaced instances (and their children) go.
	testNamespace = "workloads"
)

var (
	kroManifest = manifestSpec{
		url:      "https://github.com/kubernetes-sigs/kro/releases/download/v" + kroVersion + "/kro-core-install-manifests.yaml",
		filename: "kro-" + kroVersion + ".yaml",
	}
	istioManifest = manifestSpec{
		url:      "https://raw.githubusercontent.com/istio/istio/" + istioVersion + "/manifests/charts/base/files/crd-all.gen.yaml",
		filename: "istio-crds-" + istioVersion + ".yaml",
	}
	promManifest = manifestSpec{
		url:      "https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v" + promOperatorVersion + "/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml",
		filename: "prometheus-servicemonitor-" + promOperatorVersion + ".yaml",
	}
)
