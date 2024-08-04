package testhelpers

import (
	"context"

	g "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyFile(ctx context.Context, k8sClient client.Client, path string, ns string) {
	res := new(unstructured.Unstructured)
	ResourceFromFile(path, res)
	res.SetNamespace(ns)
	g.Expect(k8sClient.Patch(
		ctx, res,
		client.Apply,
		client.FieldOwner("test-suite"), client.ForceOwnership,
	)).To(g.Succeed())
}
