package eventr_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Eventr Suite")
}
