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
					applyFile(ctx, k8sClient, resFile, testNs)
				}
			})

			It("should successfully generate the expected resource from the template", func() {
				var err error

				controllerReconciler := &ProjectDevelopmentStreamReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}

				By("Setting the owner reference")
				Expect(ownership.HasProductRef(k8sClient, getPDS(ctx, k8sClient, testNsN))).To(BeFalse())
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
				Expect(err).NotTo(HaveOccurred())
				Expect(ownership.HasProductRef(k8sClient, getPDS(ctx, k8sClient, testNsN))).To(BeTrue())

				By("Creating the templates objects")
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: testNsN})
				Expect(err).NotTo(HaveOccurred())
				checkExpectedFile(ctx, k8sClient, expFile, testNs)
			})
		},
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
	)
})

func resourceFromFile(fname string, resource client.Object) {
	testhelpers.ResourceFromFile(
		filepath.Join("..", "..", "config", "samples", fname),
		resource,
	)
}

func applyFile(ctx context.Context, k8sClient client.Client, fname string, ns string) {
	var res unstructured.Unstructured
	resourceFromFile(fname, &res)
	res.SetNamespace(ns)
	Expect(k8sClient.Create(ctx, &res)).To(Succeed())
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
		Expect(k8sClient.Delete(ctx, &ns)).To(Succeed())
		// Add a random number to the name to make a unique NS name so we don't
		// have to wait for the deletion to finish
		nsName = fmt.Sprintf("test-ns-%d-%d", GinkgoParallelProcess(), rand.Intn(10000))
	}
	ns = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	Expect(k8sClient.Create(ctx, &ns)).To(Succeed())
	DeferCleanup(k8sClient.Delete, &ns)
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

		Expect(existing.Object).To(Equal(resource.Object))
	}
}

func dropUncomparableMetadata(obj *unstructured.Unstructured) {
	emd, ok, err := unstructured.NestedFieldCopy(obj.Object, "metadata")
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeTrue())

	emdm := emd.(map[string]interface{})
	nmd := make(map[string]interface{})
	for _, key := range []string{"ownerReferences", "annotations", "labels", "name", "namespace"} {
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
