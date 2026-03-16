package ownership_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konflux-ci/project-controller/internal/ownership"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SetWithoutUid", func() {
	var resource *corev1.ConfigMap

	BeforeEach(func() {
		resource = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource",
				Namespace: "default",
			},
		}
	})

	Context("adding first owner reference", func() {
		It("should add owner reference with no flags", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "my-deployment", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].APIVersion).To(Equal("apps/v1"))
			Expect(resource.OwnerReferences[0].Kind).To(Equal("Deployment"))
			Expect(resource.OwnerReferences[0].Name).To(Equal("my-deployment"))
			Expect(resource.OwnerReferences[0].Controller).To(BeNil())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).To(BeNil())
		})

		It("should add owner reference with Controller=true", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "my-deployment", true, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Controller).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].Controller).To(BeTrue())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).To(BeNil())
		})

		It("should add owner reference with BlockOwnerDeletion=true", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "my-deployment", false, true)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Controller).To(BeNil())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
		})

		It("should add owner reference with both flags set", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "my-deployment", true, true)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Controller).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].Controller).To(BeTrue())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
		})

		It("should handle core API group (empty group)", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "my-config", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].APIVersion).To(Equal("v1"))
			Expect(resource.OwnerReferences[0].Kind).To(Equal("ConfigMap"))
			Expect(resource.OwnerReferences[0].Name).To(Equal("my-config"))
		})

		It("should handle custom API group", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "custom.example.com",
				Version: "v1beta1",
				Kind:    "CustomResource",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "my-custom", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].APIVersion).To(Equal("custom.example.com/v1beta1"))
			Expect(resource.OwnerReferences[0].Kind).To(Equal("CustomResource"))
		})

		It("should handle long owner names", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			longName := "very-long-deployment-name-with-many-characters-that-might-exceed-normal-lengths"
			ownership.SetWithoutUid(resource, ownerGVK, longName, false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal(longName))
		})

		It("should handle empty owner name", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal(""))
		})
	})

	Context("updating existing owner reference", func() {
		BeforeEach(func() {
			// Pre-populate with an existing owner reference
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "original-deployment",
					UID:        types.UID("original-uid"),
				},
			}
		})

		It("should replace reference with same Group and Kind", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "new-deployment", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("new-deployment"))
			Expect(resource.OwnerReferences[0].UID).To(BeEmpty())
		})

		It("should update with different version but same Group and Kind", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1beta1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "new-deployment", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].APIVersion).To(Equal("apps/v1beta1"))
			Expect(resource.OwnerReferences[0].Name).To(Equal("new-deployment"))
		})

		It("should update Controller flag on existing reference", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "deployment-with-controller", true, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Controller).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].Controller).To(BeTrue())
		})

		It("should update BlockOwnerDeletion flag on existing reference", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "deployment-with-block", false, true)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
		})

		It("should update both flags on existing reference", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "deployment-with-both", true, true)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Controller).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].Controller).To(BeTrue())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
		})

		It("should clear flags when updating to false", func() {
			// First set with flags true
			trueVal := true
			resource.OwnerReferences[0].Controller = &trueVal
			resource.OwnerReferences[0].BlockOwnerDeletion = &trueVal

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "deployment-no-flags", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Controller).To(BeNil())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).To(BeNil())
		})

		It("should preserve name when updating flags only", func() {
			resource.OwnerReferences[0].Name = "important-deployment"

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "important-deployment", true, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("important-deployment"))
			Expect(resource.OwnerReferences[0].Controller).NotTo(BeNil())
		})

		It("should handle update with core API group", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "original-config",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "new-config", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("new-config"))
			Expect(resource.OwnerReferences[0].APIVersion).To(Equal("v1"))
		})

		It("should handle sequential updates", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}

			ownership.SetWithoutUid(resource, ownerGVK, "first-update", true, false)
			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("first-update"))

			ownership.SetWithoutUid(resource, ownerGVK, "second-update", false, true)
			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("second-update"))
			Expect(resource.OwnerReferences[0].Controller).To(BeNil())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).NotTo(BeNil())
		})
	})

	Context("handling multiple owner references", func() {
		BeforeEach(func() {
			// Pre-populate with multiple different owner references
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "deployment-owner",
				},
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "configmap-owner",
				},
			}
		})

		It("should preserve references with different Groups", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "batch",
				Version: "v1",
				Kind:    "Job",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "job-owner", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(3))
			// Verify original references are preserved
			hasDeployment := false
			hasConfigMap := false
			hasJob := false
			for _, ref := range resource.OwnerReferences {
				if ref.Kind == "Deployment" && ref.Name == "deployment-owner" {
					hasDeployment = true
				}
				if ref.Kind == "ConfigMap" && ref.Name == "configmap-owner" {
					hasConfigMap = true
				}
				if ref.Kind == "Job" && ref.Name == "job-owner" {
					hasJob = true
				}
			}
			Expect(hasDeployment).To(BeTrue())
			Expect(hasConfigMap).To(BeTrue())
			Expect(hasJob).To(BeTrue())
		})

		It("should preserve references with different Kinds", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "StatefulSet",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "statefulset-owner", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(3))
			// Original Deployment and ConfigMap should be preserved
			hasDeployment := false
			hasStatefulSet := false
			for _, ref := range resource.OwnerReferences {
				if ref.Kind == "Deployment" {
					hasDeployment = true
				}
				if ref.Kind == "StatefulSet" {
					hasStatefulSet = true
				}
			}
			Expect(hasDeployment).To(BeTrue())
			Expect(hasStatefulSet).To(BeTrue())
		})

		It("should update only matching Group and Kind", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "updated-deployment", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			// ConfigMap should be preserved, Deployment should be updated
			hasUpdatedDeployment := false
			hasConfigMap := false
			for _, ref := range resource.OwnerReferences {
				if ref.Kind == "Deployment" && ref.Name == "updated-deployment" {
					hasUpdatedDeployment = true
				}
				if ref.Kind == "ConfigMap" && ref.Name == "configmap-owner" {
					hasConfigMap = true
				}
			}
			Expect(hasUpdatedDeployment).To(BeTrue())
			Expect(hasConfigMap).To(BeTrue())
		})

		It("should add new non-matching reference", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "networking.k8s.io",
				Version: "v1",
				Kind:    "Ingress",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "ingress-owner", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(3))
		})

		It("should handle three different owner types", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "deployment-1",
				},
				{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Name:       "job-1",
				},
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       "service-1",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "batch",
				Version: "v1",
				Kind:    "Job",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "job-updated", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(3))
			// Verify Job was updated, others preserved
			hasDeployment := false
			hasUpdatedJob := false
			hasService := false
			for _, ref := range resource.OwnerReferences {
				if ref.Kind == "Deployment" && ref.Name == "deployment-1" {
					hasDeployment = true
				}
				if ref.Kind == "Job" && ref.Name == "job-updated" {
					hasUpdatedJob = true
				}
				if ref.Kind == "Service" && ref.Name == "service-1" {
					hasService = true
				}
			}
			Expect(hasDeployment).To(BeTrue())
			Expect(hasUpdatedJob).To(BeTrue())
			Expect(hasService).To(BeTrue())
		})

		It("should handle references with same Kind but different Groups", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "custom.io/v1",
					Kind:       "Resource",
					Name:       "custom-resource",
				},
				{
					APIVersion: "other.io/v1",
					Kind:       "Resource",
					Name:       "other-resource",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "custom.io",
				Version: "v1",
				Kind:    "Resource",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "updated-custom-resource", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			// Verify only custom.io Resource was updated
			hasUpdatedCustom := false
			hasOther := false
			for _, ref := range resource.OwnerReferences {
				if ref.APIVersion == "custom.io/v1" && ref.Name == "updated-custom-resource" {
					hasUpdatedCustom = true
				}
				if ref.APIVersion == "other.io/v1" && ref.Name == "other-resource" {
					hasOther = true
				}
			}
			Expect(hasUpdatedCustom).To(BeTrue())
			Expect(hasOther).To(BeTrue())
		})

		It("should preserve all references when adding with different Group/Kind combo", func() {
			initialCount := len(resource.OwnerReferences)

			ownerGVK := schema.GroupVersionKind{
				Group:   "custom.example.com",
				Version: "v1alpha1",
				Kind:    "CustomController",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "custom-controller", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(initialCount + 1))
		})

		It("should handle empty owner references list", func() {
			resource.OwnerReferences = []metav1.OwnerReference{}

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "first-owner", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("first-owner"))
		})

		It("should handle nil owner references list", func() {
			resource.OwnerReferences = nil

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "first-owner", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("first-owner"))
		})
	})

	Context("edge cases and special scenarios", func() {
		It("should handle various APIVersion formats", func() {
			testCases := []struct {
				gvk      schema.GroupVersionKind
				expected string
			}{
				{
					gvk:      schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
					expected: "v1",
				},
				{
					gvk:      schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
					expected: "apps/v1",
				},
				{
					gvk:      schema.GroupVersionKind{Group: "custom.io", Version: "v1beta1", Kind: "Custom"},
					expected: "custom.io/v1beta1",
				},
				{
					gvk:      schema.GroupVersionKind{Group: "very.long.domain.example.com", Version: "v2alpha3", Kind: "Resource"},
					expected: "very.long.domain.example.com/v2alpha3",
				},
			}

			for _, tc := range testCases {
				resource.OwnerReferences = nil
				ownership.SetWithoutUid(resource, tc.gvk, "test-owner", false, false)
				Expect(resource.OwnerReferences).To(HaveLen(1))
				Expect(resource.OwnerReferences[0].APIVersion).To(Equal(tc.expected))
			}
		})

		It("should handle multiple updates to same reference", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}

			ownership.SetWithoutUid(resource, ownerGVK, "owner-1", false, false)
			Expect(resource.OwnerReferences).To(HaveLen(1))

			ownership.SetWithoutUid(resource, ownerGVK, "owner-2", true, false)
			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("owner-2"))

			ownership.SetWithoutUid(resource, ownerGVK, "owner-3", false, true)
			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("owner-3"))
		})

		It("should correctly match Group when APIVersion has different versions", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1beta1",
					Kind:       "Deployment",
					Name:       "old-deployment",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "new-deployment", false, false)

			// Should update because Group and Kind match, even with different versions
			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal("new-deployment"))
			Expect(resource.OwnerReferences[0].APIVersion).To(Equal("apps/v1"))
		})

		It("should not match when Group differs", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "apps-deployment",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "extensions",
				Version: "v1beta1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "extensions-deployment", false, false)

			// Should add new reference because Group differs
			Expect(resource.OwnerReferences).To(HaveLen(2))
		})

		It("should handle invalid APIVersion in existing reference gracefully", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "invalid-api-version-format",
					Kind:       "Deployment",
					Name:       "invalid-owner",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "valid-owner", false, false)

			// Should add new reference because existing one has invalid APIVersion
			Expect(resource.OwnerReferences).To(HaveLen(2))
		})

		It("should preserve order when updating middle reference", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
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
				{
					APIVersion: "batch/v1",
					Kind:       "Job",
					Name:       "job-1",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			ownership.SetWithoutUid(resource, ownerGVK, "deploy-updated", false, false)

			Expect(resource.OwnerReferences).To(HaveLen(3))
			Expect(resource.OwnerReferences[0].Kind).To(Equal("ConfigMap"))
			Expect(resource.OwnerReferences[1].Kind).To(Equal("Deployment"))
			Expect(resource.OwnerReferences[1].Name).To(Equal("deploy-updated"))
			Expect(resource.OwnerReferences[2].Kind).To(Equal("Job"))
		})

		It("should work with different K8s resource types", func() {
			// Test with different resource types to ensure it works universally
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			}

			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "ReplicaSet",
			}
			ownership.SetWithoutUid(pod, ownerGVK, "my-replicaset", true, false)

			Expect(pod.OwnerReferences).To(HaveLen(1))
			Expect(pod.OwnerReferences[0].Kind).To(Equal("ReplicaSet"))
			Expect(pod.OwnerReferences[0].Name).To(Equal("my-replicaset"))
		})

		It("should handle special characters in names", func() {
			ownerGVK := schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}
			specialName := "my-deployment-123_test.example"
			ownership.SetWithoutUid(resource, ownerGVK, specialName, false, false)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].Name).To(Equal(specialName))
		})
	})
})
