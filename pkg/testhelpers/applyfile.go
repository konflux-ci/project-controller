package testhelpers

import (
	"context"

	g "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ApplyFile(ctx context.Context, k8sClient client.Client, path string, ns string) {
	var res unstructured.Unstructured
	ResourceFromFile(path, &res)
	res.SetNamespace(ns)
	g.Expect(k8sClient.Create(ctx, &res)).To(g.Succeed())
}
