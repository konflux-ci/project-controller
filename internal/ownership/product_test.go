package ownership_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konflux-ci/project-controller/internal/ownership"
	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("HasProductRef", func() {
	var (
		fakeClient client.Client
		scheme     *runtime.Scheme
		pds        projctlv1beta1.ProjectDevelopmentStream
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		_ = projctlv1beta1.AddToScheme(scheme)
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		pds = projctlv1beta1.ProjectDevelopmentStream{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pds",
				Namespace: "default",
			},
			Spec: projctlv1beta1.ProjectDevelopmentStreamSpec{
				Project: "my-project",
			},
		}
	})

	Context("when project field is empty", func() {
		It("should return true", func() {
			pds.Spec.Project = ""
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeTrue())
		})
	})

	Context("when valid project reference exists", func() {
		BeforeEach(func() {
			projectGVK, _ := fakeClient.GroupVersionKindFor(&projctlv1beta1.Project{})
			apiVersion, kind := projectGVK.ToAPIVersionAndKind()

			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "my-project",
				},
			}
		})

		It("should return true", func() {
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeTrue())
		})
	})

	Context("when project reference is missing", func() {
		It("should return false when no owner references exist", func() {
			pds.OwnerReferences = []metav1.OwnerReference{}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})

		It("should return false when owner references exist but none match", func() {
			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "some-config",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})
	})

	Context("when project reference has wrong APIVersion", func() {
		It("should return false", func() {
			projectGVK, _ := fakeClient.GroupVersionKindFor(&projctlv1beta1.Project{})
			_, kind := projectGVK.ToAPIVersionAndKind()

			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "wrong.api/v1",
					Kind:       kind,
					Name:       "my-project",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})
	})

	Context("when project reference has wrong Kind", func() {
		It("should return false", func() {
			projectGVK, _ := fakeClient.GroupVersionKindFor(&projctlv1beta1.Project{})
			apiVersion, _ := projectGVK.ToAPIVersionAndKind()

			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: apiVersion,
					Kind:       "WrongKind",
					Name:       "my-project",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})
	})

	Context("when project reference has wrong Name", func() {
		It("should return false", func() {
			projectGVK, _ := fakeClient.GroupVersionKindFor(&projctlv1beta1.Project{})
			apiVersion, kind := projectGVK.ToAPIVersionAndKind()

			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "different-project",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})
	})

	Context("when multiple owner references exist", func() {
		It("should return true when correct reference is present among others", func() {
			projectGVK, _ := fakeClient.GroupVersionKindFor(&projctlv1beta1.Project{})
			apiVersion, kind := projectGVK.ToAPIVersionAndKind()

			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "some-config",
				},
				{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "my-project",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "some-deployment",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeTrue())
		})

		It("should return false when all references are wrong", func() {
			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "config-1",
				},
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "deploy-1",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})
	})

	Context("edge cases", func() {
		It("should handle nil owner references", func() {
			pds.OwnerReferences = nil
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})

		It("should handle partial matches correctly", func() {
			projectGVK, _ := fakeClient.GroupVersionKindFor(&projctlv1beta1.Project{})
			apiVersion, kind := projectGVK.ToAPIVersionAndKind()

			// Correct APIVersion and Kind but wrong Name
			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: apiVersion,
					Kind:       kind,
					Name:       "wrong-name",
				},
			}
			result := ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())

			// Correct Kind and Name but wrong APIVersion
			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "wrong/v1",
					Kind:       kind,
					Name:       "my-project",
				},
			}
			result = ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())

			// Correct APIVersion and Name but wrong Kind
			pds.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: apiVersion,
					Kind:       "WrongKind",
					Name:       "my-project",
				},
			}
			result = ownership.HasProductRef(fakeClient, pds)
			Expect(result).To(BeFalse())
		})
	})
})
