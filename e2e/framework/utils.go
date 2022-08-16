package framework

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func RenderManifest(manifest string, data any) (string, error) {
	temp, err := template.New("manifest").Parse(manifest)
	if err != nil {
		return "", err
	}

	artifact := new(strings.Builder)
	if err := temp.Execute(artifact, data); err != nil {
		return "", err
	}
	return artifact.String(), nil
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

// GetKubeConfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeConfig() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeConfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeConfig); err != nil && !os.IsNotExist(err) {
			kubeConfig = ""
		}
	}
	return kubeConfig
}

func randomNamespace() string {
	return fmt.Sprintf("amesh-e2e-%d", time.Now().Nanosecond())
}
