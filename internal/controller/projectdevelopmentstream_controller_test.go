/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/internal/ownership"
	"github.com/konflux-ci/project-controller/pkg/testhelpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("ProjectDevelopmentStream Controller", func() {
	DescribeTableSubtree("When reconciling a PDS resource",
		func(pdsName, expFile string, resFiles ...string) {
			ctx := context.Background()

			var testNs string
			var testNsN types.NamespacedName

			BeforeEach(func() {
				testNs = setupTestNamespace(ctx, k8sClient)
				testNsN = types.NamespacedName{Namespace: testNs, Name: pdsName}

				for _, resFile := range resFiles {
					applySampleFile(ctx, k8sClient, resFile, testNs)
				}
			})

			It("should successfully generate the expected resource from the template", func() {
				var err error

				controllerReconciler := &ProjectDevelopmentStreamReconciler{
					Client:   saClient,
					Scheme:   saClient.Scheme(),
					Recorder: saCluster.GetEventRecorder("ProjectDevelopmentStream-controller-tests"),
				}

				By("Setting the owner reference")
				Expect(ownership.HasProductRef(k8sClient, getPDS(ctx, k8sClient, testNsN))).To(BeFalse())
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
				Expect(err).NotTo(HaveOccurred())
				Expect(ownership.HasProductRef(k8sClient, getPDS(ctx, k8sClient, testNsN))).To(BeTrue())

				By("Verifying UpdatingOwnerRef status after first reconcile")
				updatedPds := getPDS(ctx, k8sClient, testNsN)
				Expect(updatedPds.Status.Conditions).To(HaveLen(1))
				Expect(updatedPds.Status.Conditions[0].Type).To(Equal(ConditionTypeReady))
				Expect(updatedPds.Status.Conditions[0].Status).To(Equal(metav1.ConditionUnknown))
				Expect(updatedPds.Status.Conditions[0].Reason).To(Equal("UpdatingOwnerRef"))

				By("Creating the templates objects")
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
				Expect(err).NotTo(HaveOccurred())
				checkExpectedFile(ctx, k8sClient, expFile, testNs)
			})
		},
		// The following 5 tests primarily verify template resource generation.
		// They also verify status condition: Ready=True, UpdatingOwnerRef (after 1st reconcile) and ResourcesApplied (final)
		Entry(
			"Application and Component resources",
			"projectdevelopmentstream-sample-w-template-vars",
			"projctl_v1beta1_pds_w_tmp_vars_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_projectdevelopmentstreamtemplate.yaml",
			"projctl_v1beta1_projectdevelopmentstream_w_template_vars.yaml",
		),
		Entry(
			"ImageRepository resource",
			"pds-sample-w-imagerepo",
			"projctl_v1beta1_pds_w_imagerepo_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pdst_w_imagerepo.yaml",
			"projctl_v1beta1_pds_w_imagerepo.yaml",
		),
		Entry(
			"IntegTestScenario resource",
			"pds-sample-w-intgtstscnario",
			"projctl_v1beta1_pds_w_intgtstscnario_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pdst_w_intgtstscnario.yaml",
			"projctl_v1beta1_pds_w_intgtstscnario.yaml",
		),
		Entry(
			"ReleasePlan resource",
			"pds-sample-w-relpln",
			"projctl_v1beta1_pds_w_relpln_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pdst_w_relpln.yaml",
			"projctl_v1beta1_pds_w_relpln.yaml",
		),
		Entry(
			"Existing Component resource",
			"pds-sample-w-existing-comp",
			"projctl_v1beta1_pds_w_existing_comp_exp_results.yaml",
			"appstudio_v1alpha1_comp.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pdst_w_existing_comp.yaml",
			"projctl_v1beta1_pds_w_existing_comp.yaml",
		),
		// Status: Ready=True, Reason: NoTemplate
		Entry(
			"No template specified",
			"pds-no-template",
			"projctl_v1beta1_pds_no_template_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pds_no_template.yaml",
		),
		// Status: Ready=False, Reason: TemplateFetchFailed
		Entry(
			"Template not found",
			"pds-template-not-found",
			"projctl_v1beta1_pds_template_not_found_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pds_template_not_found.yaml",
		),
		// Status: Ready=False, Reason: TemplateGenerationFailed
		Entry(
			"Invalid template syntax",
			"pds-invalid-template",
			"projctl_v1beta1_pds_invalid_template_exp_results.yaml",
			"projctl_v1beta1_project.yaml",
			"projctl_v1beta1_pdst_invalid_template.yaml",
			"projctl_v1beta1_pds_invalid_template.yaml",
		),
		// Note: The following status reasons are NOT tested:
		// - "Reconciling": Too transient, immediately overwritten by subsequent conditions
		// - "ApplyingResources": Requires reliable resource conflict simulation, difficult to test without flakiness
	)
})

var _ = Describe("checkProductOwnerRef", func() {
	var reconciler *ProjectDevelopmentStreamReconciler

	BeforeEach(func() {
		reconciler = &ProjectDevelopmentStreamReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	DescribeTable(
		"reports whether the PDS already has the correct product owner reference",
		func(pds projctlv1beta1.ProjectDevelopmentStream, expected bool) {
			Expect(reconciler.checkProductOwnerRef(pds)).To(Equal(expected))
		},
		Entry("empty project", projctlv1beta1.ProjectDevelopmentStream{}, true),
		Entry(
			"project set without owner reference",
			projctlv1beta1.ProjectDevelopmentStream{
				Spec: projctlv1beta1.ProjectDevelopmentStreamSpec{Project: "project-sample"},
			},
			false,
		),
	)
})

func applySampleFile(ctx context.Context, k8sClient client.Client, fname string, ns string) {
	testhelpers.ApplyFile(
		ctx, k8sClient,
		filepath.Join("..", "..", "config", "samples", fname),
		ns,
	)
}

func setupTestNamespace(ctx context.Context, k8sClient client.Client) string {
	var ns corev1.Namespace
	nsName := fmt.Sprintf("test-ns-%d", GinkgoParallelProcess())
	for {
		nsNsName := types.NamespacedName{
			Name:      nsName,
			Namespace: "default",
		}
		err := k8sClient.Get(ctx, nsNsName, &ns)
		if errors.IsNotFound(err) {
			break
		}
		Expect(err).NotTo(HaveOccurred())
		if !keepNamespaces() {
			Expect(k8sClient.Delete(ctx, &ns)).To(Succeed())
		}
		// Add a random number to the name to make a unique NS name so we don't
		// have to wait for the deletion to finish
		nsName = fmt.Sprintf("test-ns-%d-%d", GinkgoParallelProcess(), rand.Intn(10000)) //nolint:gosec // test namespace uniqueness
	}
	ns = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	Expect(k8sClient.Create(ctx, &ns)).To(Succeed())
	if !keepNamespaces() {
		DeferCleanup(k8sClient.Delete, &ns)
	}
	return nsName
}

func getPDS(ctx context.Context, k8sClient client.Client, nsn types.NamespacedName) projctlv1beta1.ProjectDevelopmentStream {
	pds := &projctlv1beta1.ProjectDevelopmentStream{}
	Expect(k8sClient.Get(ctx, nsn, pds)).To(Succeed())
	return *pds
}

func checkExpectedFile(ctx context.Context, k8sClient client.Client, fname string, ns string) {
	f, err := os.Open(filepath.Join("..", "..", "config", "samples", fname))
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = f.Close() }()
	buf, err := io.ReadAll(f)
	Expect(err).NotTo(HaveOccurred())
	for _, subBuf := range bytes.Split(buf, []byte("---\n")) {
		var resource unstructured.Unstructured
		Expect(yaml.UnmarshalStrict(subBuf, &resource)).To(Succeed())
		resource.SetNamespace(ns)

		var existing unstructured.Unstructured
		existing.SetAPIVersion(resource.GetAPIVersion())
		existing.SetKind(resource.GetKind())
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&resource), &existing)).To(Succeed())

		dropUncomparableMetadata(&existing)
		dropTimestamps(&existing)
		dropTimestamps(&resource)

		Expect(existing.Object).To(Equal(resource.Object))
	}
}

func dropUncomparableMetadata(obj *unstructured.Unstructured) {
	emd, ok, err := unstructured.NestedFieldCopy(obj.Object, "metadata")
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeTrue())

	emdm := emd.(map[string]interface{})
	nmd := make(map[string]interface{})
	for _, key := range []string{"ownerReferences", "annotations", "labels", "name", "namespace", "finalizers"} {
		if v, ok := emdm[key]; ok {
			nmd[key] = v
		}
	}
	if o, ok := nmd["ownerReferences"]; ok {
		ol := o.([]interface{})
		for _, oi := range ol {
			om := oi.(map[string]interface{})
			delete(om, "uid")
		}
	}
	Expect(unstructured.SetNestedField(obj.Object, nmd, "metadata")).To(Succeed())
}

func dropTimestamps(obj *unstructured.Unstructured) {
	// Normalize lastTransitionTime to a fixed value for comparison
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return
	}

	for i := range conditions {
		if condition, ok := conditions[i].(map[string]interface{}); ok {
			if _, hasTimestamp := condition["lastTransitionTime"]; hasTimestamp {
				condition["lastTransitionTime"] = "1970-01-01T00:00:00Z"
			}
		}
	}

	Expect(unstructured.SetNestedSlice(obj.Object, conditions, "status", "conditions")).To(Succeed())
}

func keepNamespaces() bool {
	return strings.ToLower(os.Getenv("KEEP_TEST_NAMESPACES")) == "true"
}

var _ = Describe("ImageRepository annotation persistence", func() {
	Context("When reconciling a PDS with ImageRepository", func() {
		var (
			ctx        context.Context
			testNs     string
			testNsN    types.NamespacedName
			reconciler *ProjectDevelopmentStreamReconciler
		)

		BeforeEach(func() {
			ctx = context.Background()
			testNs = setupTestNamespace(ctx, k8sClient)
			testNsN = types.NamespacedName{
				Namespace: testNs,
				Name:      "pds-sample-w-imagerepo",
			}

			// Apply test resources
			applySampleFile(ctx, k8sClient, "projctl_v1beta1_project.yaml", testNs)
			applySampleFile(ctx, k8sClient, "projctl_v1beta1_pdst_w_imagerepo.yaml", testNs)
			applySampleFile(ctx, k8sClient, "projctl_v1beta1_pds_w_imagerepo.yaml", testNs)

			reconciler = &ProjectDevelopmentStreamReconciler{
				Client:   saClient,
				Scheme:   saClient.Scheme(),
				Recorder: saCluster.GetEventRecorder("ProjectDevelopmentStream-controller-tests"),
			}
		})

		It("should preserve image-controller annotation across reconciliations", func() {
			// First reconcile: Set owner reference
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: Create resources including ImageRepository with annotation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify ImageRepository was created with the annotation
			imageRepoName := types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-repo-2-2-0",
			}
			imageRepo := &unstructured.Unstructured{}
			imageRepo.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			imageRepo.SetKind("ImageRepository")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			// The annotation should be present after initial creation
			annotations := imageRepo.GetAnnotations()
			Expect(annotations).NotTo(BeNil())
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation))
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal("true"))

			// Third reconcile: Should preserve the annotation (live resource has it)
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify the annotation is still present
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(BeNil())
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation),
				"image-controller annotation should persist across reconciliations")
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal("true"))
		})

		It("should NOT re-apply annotation after image-controller removes it", func() {
			// First reconcile: Set owner reference
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: Create resources including ImageRepository with annotation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify ImageRepository was created with the annotation
			imageRepoName := types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-repo-2-2-0",
			}
			imageRepo := &unstructured.Unstructured{}
			imageRepo.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			imageRepo.SetKind("ImageRepository")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			annotations := imageRepo.GetAnnotations()
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation))

			// Simulate image-controller processing by removing the annotation
			delete(annotations, ImageControllerUpdateAnnotation)
			imageRepo.SetAnnotations(annotations)
			err = k8sClient.Update(ctx, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			// Verify annotation was removed
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(ImageControllerUpdateAnnotation),
				"annotation should be removed to simulate image-controller processing")

			// Third reconcile: project-controller should NOT restore the annotation
			// because the live ImageRepository doesn't have it (image-controller removed it)
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify the annotation is still absent (not restored)
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(ImageControllerUpdateAnnotation),
				"project-controller should NOT restore annotation after image-controller removes it")
		})

		It("REGRESSION: should apply annotation even when Component has templated containerImage", func() {
			// This tests Yftach's regression: if Component is created with containerImage
			// already set from the template, we should STILL apply the annotation on
			// ImageRepository's first creation.
			// With the new logic (checking live ImageRepository), this should now PASS.

			// Manually create Component with containerImage already set (simulating template)
			component := &unstructured.Unstructured{}
			component.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			component.SetKind("Component")
			component.SetNamespace(testNs)
			component.SetName("cool-comp1-2-2-0")

			// Set containerImage BEFORE ImageRepository is created (like a template would)
			err := unstructured.SetNestedField(component.Object, "cool-app-2-2-0", "spec", "application")
			Expect(err).NotTo(HaveOccurred())
			err = unstructured.SetNestedField(component.Object, "quay.io/pre-set/image:v1", "spec", "containerImage")
			Expect(err).NotTo(HaveOccurred())
			err = unstructured.SetNestedMap(component.Object, map[string]interface{}{
				"git": map[string]interface{}{
					"url": "https://github.com/example/repo",
				},
			}, "spec", "source")
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Create(ctx, component)
			Expect(err).NotTo(HaveOccurred())

			// First reconcile: Set owner reference
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: Create ImageRepository
			// At this point, Component.spec.containerImage is ALREADY set
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify ImageRepository was created WITH the annotation
			// (This PASSES with the new logic because we check live ImageRepository, not Component)
			imageRepoName := types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-repo-2-2-0",
			}
			imageRepo := &unstructured.Unstructured{}
			imageRepo.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			imageRepo.SetKind("ImageRepository")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			annotations := imageRepo.GetAnnotations()
			Expect(annotations).NotTo(BeNil(), "ImageRepository should have annotations")
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation),
				"annotation MUST be present even though Component already has containerImage from template")
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal("true"))
		})

		It("should treat empty-string annotation as removed by image-controller", func() {
			// Test case 1 from Yftach: empty-string annotation should be treated the same as absent.
			// If image-controller sets update-component-image: "" instead of deleting it,
			// project-controller should NOT re-apply it from the template.

			// First reconcile: Set owner reference
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: Create resources including ImageRepository with annotation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify ImageRepository was created
			imageRepoName := types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-repo-2-2-0",
			}
			imageRepo := &unstructured.Unstructured{}
			imageRepo.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			imageRepo.SetKind("ImageRepository")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			// Simulate image-controller setting the annotation to empty string instead of deleting it
			annotations := imageRepo.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[ImageControllerUpdateAnnotation] = ""
			imageRepo.SetAnnotations(annotations)
			err = k8sClient.Update(ctx, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			// Verify annotation is now empty string
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal(""),
				"annotation should be set to empty string to simulate image-controller behavior variant")

			// Third reconcile: project-controller should NOT restore the annotation to "true"
			// because empty string is treated as "processed" (same as absent)
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify the annotation is still empty (not restored to "true")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal(""),
				"project-controller should NOT change empty-string annotation back to 'true'")
		})

		It("should NOT restore annotation after premature loss before image-controller processes it", func() {
			// Test case 2 from Yftach: if the annotation is lost (deleted by an external actor or bug)
			// before image-controller processes it, project-controller should NOT restore it.
			// This documents the current no-self-heal behavior and prevents accidental churn reintroduction.

			// First reconcile: Set owner reference
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile: Create resources including ImageRepository with annotation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify ImageRepository was created with the annotation
			imageRepoName := types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-repo-2-2-0",
			}
			imageRepo := &unstructured.Unstructured{}
			imageRepo.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			imageRepo.SetKind("ImageRepository")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			annotations := imageRepo.GetAnnotations()
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation))
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal("true"))

			// Simulate premature annotation loss (e.g., external actor, bug, or race)
			// This happens BEFORE image-controller has a chance to process it
			delete(annotations, ImageControllerUpdateAnnotation)
			imageRepo.SetAnnotations(annotations)
			err = k8sClient.Update(ctx, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			// Verify annotation was removed
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(ImageControllerUpdateAnnotation),
				"annotation should be absent to simulate premature loss")

			// Third reconcile: project-controller should NOT restore the annotation
			// This is the documented behavior: no self-healing to avoid churn
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify the annotation is still absent (not restored from template)
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(ImageControllerUpdateAnnotation),
				"project-controller should NOT restore annotation after premature loss (no self-heal)")
		})

		It("should handle full bootstrap lifecycle with sequential reconciles", func() {
			// Test case 4 from Yftach (optional): end-to-end bootstrap sequence
			// This consolidates the happy path into a single spec to catch regressions
			// where one step breaks in isolation.

			imageRepoName := types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-repo-2-2-0",
			}

			// Step 1: First reconcile - Set owner reference
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Step 2: Second reconcile - Create ImageRepository with annotation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify: ImageRepository created with annotation
			imageRepo := &unstructured.Unstructured{}
			imageRepo.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			imageRepo.SetKind("ImageRepository")
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations := imageRepo.GetAnnotations()
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation))
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal("true"),
				"annotation should be present after bootstrap")

			// Step 3: Third reconcile - Annotation still present (image-controller hasn't processed yet)
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Verify: Annotation preserved across reconciles
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).To(HaveKey(ImageControllerUpdateAnnotation))
			Expect(annotations[ImageControllerUpdateAnnotation]).To(Equal("true"),
				"annotation should persist while image-controller hasn't acted")

			// Step 4: Simulate image-controller completion (remove annotation, set containerImage on Component)
			delete(annotations, ImageControllerUpdateAnnotation)
			imageRepo.SetAnnotations(annotations)
			err = k8sClient.Update(ctx, imageRepo)
			Expect(err).NotTo(HaveOccurred())

			// Also update Component to simulate full image-controller workflow
			component := &unstructured.Unstructured{}
			component.SetAPIVersion("appstudio.redhat.com/v1alpha1")
			component.SetKind("Component")
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: testNs,
				Name:      "cool-comp1-2-2-0",
			}, component)
			if err == nil {
				_ = unstructured.SetNestedField(component.Object, "quay.io/konflux/cool-comp1:latest", "spec", "containerImage")
				_ = k8sClient.Update(ctx, component)
			}

			// Verify: Annotation removed
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(ImageControllerUpdateAnnotation),
				"annotation should be absent after image-controller processing")

			// Step 5: Fourth reconcile - Should NOT restore annotation
			_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
			Expect(err).NotTo(HaveOccurred())

			// Final verification: Annotation stays absent
			err = k8sClient.Get(ctx, imageRepoName, imageRepo)
			Expect(err).NotTo(HaveOccurred())
			annotations = imageRepo.GetAnnotations()
			Expect(annotations).NotTo(HaveKey(ImageControllerUpdateAnnotation),
				"annotation should NOT be restored after image-controller completes")
		})
	})
})
