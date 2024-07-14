package ownership

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
)

func HasProductRef(client client.Client, pds projctlv1beta1.ProjectDevelopmentStream) bool {
	projectName := pds.Spec.Project
	if projectName == "" {
		return true // We define an empty project field as having a reference
	}
	projectGVK, _ := client.GroupVersionKindFor(&projctlv1beta1.Project{})
	prjAPIVersion, prjKind := projectGVK.ToAPIVersionAndKind()
	for _, ref := range pds.ObjectMeta.OwnerReferences {
		if ref.APIVersion == prjAPIVersion && ref.Kind == prjKind && ref.Name == projectName {
			return true
		}
	}
	return false

}
