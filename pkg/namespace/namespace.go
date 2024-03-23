// Utils for namespace operations

package namespace

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenerateNamespaceName(crdName string, prefix string) string {
	return prefix + crdName
}

func DeleteNamespace(ctx context.Context, k8sClient client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return k8sClient.Delete(ctx, ns)
}

func CreateNamespace(ctx context.Context, k8sClient client.Client, ns *corev1.Namespace) error {
	return k8sClient.Create(ctx, ns)
}

// Checks if a string is present in a slice
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Remove a string from a slice
func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item != s {
			result = append(result, s)
		}
	}
	return
}

// Convert a map[string]string into strings
func MapToStrings(m map[string]string) string {
	var result []string
	for key, value := range m {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(result, ", ")
}

// Compare maps and give priority to values in map1 (CRD) than in map2 (NS).
// Keeps labels present in map2 but not in map1.
func MergeMaps(map1 map[string]string, map2 map[string]string) map[string]string {
	diff := make(map[string]string)
	for key1, value1 := range map1 {
		diff[key1] = value1
	}
	for key2, value2 := range map2 {
		if _, exists := diff[key2]; !exists &&
			strings.Contains(key2, "kubernetes.io") {
			diff[key2] = value2
		}
	}
	return diff
}

// Takes ns name and strips a prefix if there is any
func DeriveNamespaceConfigNameFromNamespace(nsName string) string {
	divided := strings.Split(nsName, "-")
	if len(divided) > 1 {
		noPrefixName := strings.Join(divided[1:], "")
		return noPrefixName
	} else {
		return nsName
	}

}
