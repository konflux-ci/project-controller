package testhelpers

import (
	"context"

	g "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyFile(ctx context.Context, k8sClient client.Client, path string, ns string) {
	var res, fileRes *unstructured.Unstructured
	fileRes = new(unstructured.Unstructured)
	ResourceFromFile(path, fileRes)
	fileRes.SetNamespace(ns)
	// Keep a clean copy of the data from the file
	res = fileRes.DeepCopy()
	err := k8sClient.Create(ctx, res)
	if !apierrors.IsAlreadyExists(err) {
		g.Expect(err).NotTo(g.HaveOccurred())
	}
	res = fileRes.DeepCopy()
	g.Expect(k8sClient.Patch(
		ctx, res,
		client.Apply,
		client.FieldOwner("test-suite"), client.ForceOwnership,
	)).To(g.Succeed())
}
