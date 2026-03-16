package ownership_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/internal/ownership"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("AddMissingUIDs", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		scheme     *runtime.Scheme
		resource   *corev1.ConfigMap
		ownerCM    *corev1.ConfigMap
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = projctlv1beta1.AddToScheme(scheme)

		// Create owner object with known UID
		ownerCM = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "owner-cm",
				Namespace: "default",
				UID:       types.UID("owner-uid-123"),
			},
		}

		// Create resource with owner ref missing UID
		resource = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "resource-cm",
				Namespace: "default",
			},
		}

		// Build fake client with owner pre-populated
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(ownerCM).
			Build()
	})

	Context("when owner references have no UIDs", func() {
		It("should fetch UIDs and populate them", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
					// UID intentionally missing
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
		})

		It("should fetch UIDs for multiple missing UIDs", func() {
			// Create additional owner
			ownerPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-pod",
					Namespace: "default",
					UID:       types.UID("owner-pod-456"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ownerCM, ownerPod).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "owner-pod",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
			Expect(resource.OwnerReferences[1].UID).To(Equal(types.UID("owner-pod-456")))
		})
	})

	Context("when owner object does not exist", func() {
		It("should leave UID empty", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "non-existent-owner",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].UID).To(BeEmpty())
		})

		It("should populate existing owners and skip non-existent ones", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "non-existent",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
			Expect(resource.OwnerReferences[1].UID).To(BeEmpty())
		})
	})

	Context("when some UIDs are already present", func() {
		It("should preserve existing UIDs", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
					UID:        types.UID("already-set-uid"),
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(1))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("already-set-uid")))
		})

		It("should populate missing UIDs while preserving existing ones", func() {
			ownerPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-pod",
					Namespace: "default",
					UID:       types.UID("owner-pod-456"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ownerCM, ownerPod).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
					UID:        types.UID("existing-uid"),
				},
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "owner-pod",
					// UID missing
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("existing-uid")))
			Expect(resource.OwnerReferences[1].UID).To(Equal(types.UID("owner-pod-456")))
		})
	})

	Context("when no owner references exist", func() {
		It("should handle empty owner references list", func() {
			resource.OwnerReferences = []metav1.OwnerReference{}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(0))
		})

		It("should handle nil owner references list", func() {
			resource.OwnerReferences = nil

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(BeNil())
		})
	})

	Context("when all UIDs are already present", func() {
		It("should not modify owner references", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
					UID:        types.UID("uid-1"),
				},
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "owner-pod",
					UID:        types.UID("uid-2"),
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("uid-1")))
			Expect(resource.OwnerReferences[1].UID).To(Equal(types.UID("uid-2")))
		})
	})

	Context("with different object kinds", func() {
		It("should handle Pod owners", func() {
			ownerPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-pod",
					Namespace: "default",
					UID:       types.UID("pod-uid-789"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ownerPod).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "owner-pod",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("pod-uid-789")))
		})

		It("should handle Service owners", func() {
			ownerService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-service",
					Namespace: "default",
					UID:       types.UID("service-uid-abc"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ownerService).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       "owner-service",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("service-uid-abc")))
		})

		It("should handle custom resource owners", func() {
			ownerProject := &projctlv1beta1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-project",
					Namespace: "default",
					UID:       types.UID("project-uid-xyz"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ownerProject).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "projctl.konflux.dev/v1beta1",
					Kind:       "Project",
					Name:       "owner-project",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("project-uid-xyz")))
		})
	})

	Context("with owners in same namespace", func() {
		It("should fetch UID for owner in same namespace", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
		})

		It("should not find owner in different namespace", func() {
			differentNsOwner := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-cm",
					Namespace: "other-namespace",
					UID:       types.UID("different-ns-uid"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(differentNsOwner).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			// Should not find owner because it's in a different namespace
			Expect(resource.OwnerReferences[0].UID).To(BeEmpty())
		})
	})

	Context("edge cases", func() {
		It("should handle multiple references to same owner", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(2))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
			Expect(resource.OwnerReferences[1].UID).To(Equal(types.UID("owner-uid-123")))
		})

		It("should handle mix of existing and missing UIDs with non-existent owners", func() {
			ownerPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-pod",
					Namespace: "default",
					UID:       types.UID("pod-uid-999"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ownerCM, ownerPod).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
					UID:        types.UID("already-set"),
				},
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "existing-pod",
					// Missing UID
				},
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       "non-existent-service",
					// Missing UID
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences).To(HaveLen(3))
			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("already-set")))
			Expect(resource.OwnerReferences[1].UID).To(Equal(types.UID("pod-uid-999")))
			Expect(resource.OwnerReferences[2].UID).To(BeEmpty())
		})

		It("should handle owner with special characters in name", func() {
			specialOwner := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "owner-with-special.chars_123",
					Namespace: "default",
					UID:       types.UID("special-uid"),
				},
			}
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(specialOwner).
				Build()

			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-with-special.chars_123",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("special-uid")))
		})

		It("should preserve other owner reference fields", func() {
			trueVal := true
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "ConfigMap",
					Name:               "owner-cm",
					Controller:         &trueVal,
					BlockOwnerDeletion: &trueVal,
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
			Expect(resource.OwnerReferences[0].Controller).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].Controller).To(BeTrue())
			Expect(resource.OwnerReferences[0].BlockOwnerDeletion).NotTo(BeNil())
			Expect(*resource.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
		})

		It("should handle invalid APIVersion gracefully", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "invalid-version",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
			}

			// Should not panic, just leave UID empty
			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(BeEmpty())
		})

		It("should handle empty owner name", func() {
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(BeEmpty())
		})
	})

	Context("context handling", func() {
		It("should work with background context", func() {
			ctx = context.Background()
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
		})

		It("should work with TODO context", func() {
			ctx = context.TODO()
			resource.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner-cm",
				},
			}

			ownership.AddMissingUIDs(ctx, fakeClient, resource)

			Expect(resource.OwnerReferences[0].UID).To(Equal(types.UID("owner-uid-123")))
		})
	})
})
