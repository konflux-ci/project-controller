package template

import (
	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func MkResources(
	pds projctlv1beta1.ProjectDevelopmentStream,
	pdst projctlv1beta1.ProjectDevelopmentStreamTemplate,
) ([]*unstructured.Unstructured, error) {
	resources := make([]*unstructured.Unstructured, 0, len(pdst.Spec.Resources))
	for _, unstructuredObj := range pdst.Spec.Resources {
		resource := unstructuredObj.Unstructured.DeepCopy()
		resource.SetNamespace(pds.GetNamespace())
		resources = append(resources, resource)
	}
	return resources, nil
}
