package ownership_test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/internal/ownership"
)

var _ = Describe("HasProductRef", func() {
	var k8sClient client.Client
	var projectAPIVersion, projectKind string

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		utilruntime.Must(projctlv1beta1.AddToScheme(scheme))
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		gvk, err := k8sClient.GroupVersionKindFor(&projctlv1beta1.Project{})
		Expect(err).NotTo(HaveOccurred())
		projectAPIVersion, projectKind = gvk.ToAPIVersionAndKind()
	})

	DescribeTable(
		"reports whether the PDS has a product owner reference",
		func(project string, ownerRefName string, withOwnerRef bool, expected bool) {
			pds := projctlv1beta1.ProjectDevelopmentStream{
				Spec: projctlv1beta1.ProjectDevelopmentStreamSpec{Project: project},
			}
			if withOwnerRef {
				pds.OwnerReferences = []metav1.OwnerReference{{
					APIVersion: projectAPIVersion,
					Kind:       projectKind,
					Name:       ownerRefName,
				}}
			}
			Expect(ownership.HasProductRef(k8sClient, pds)).To(Equal(expected))
		},
		Entry("empty project", "", "", false, true),
		Entry("no owner reference", "my-project", "", false, false),
		Entry("matching owner reference", "my-project", "my-project", true, true),
		Entry("wrong owner reference name", "my-project", "other-project", true, false),
	)
})
