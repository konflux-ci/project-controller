package testhelpers

import (
	"io"
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	g "github.com/onsi/gomega"
)

func ResourceFromFile(path string, resource client.Object) {
	f, err := os.Open(path)
	g.Expect(err).NotTo(g.HaveOccurred())
	defer func() { _ = f.Close() }()
	buf, err := io.ReadAll(f)
	g.Expect(err).NotTo(g.HaveOccurred())
	g.Expect(yaml.UnmarshalStrict(buf, resource)).To(g.Succeed())
}
