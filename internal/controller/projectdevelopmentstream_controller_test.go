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
					Recorder: saCluster.GetEventRecorderFor("ProjectDevelopmentStream-controller-tests"),
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
		nsName = fmt.Sprintf("test-ns-%d-%d", GinkgoParallelProcess(), rand.Intn(10000))
	}
	ns = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	Expect(k8sClient.Create(ctx, &ns)).To(Succeed())
	if !keepNamespaces() {
		DeferCleanup(k8sClient.Delete, &ns)
	}
	return nsName
}

func getPDS(ctx context.Context, client client.Client, nsn types.NamespacedName) projctlv1beta1.ProjectDevelopmentStream {
	pds := &projctlv1beta1.ProjectDevelopmentStream{}
	Expect(client.Get(ctx, nsn, pds)).To(Succeed())
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
