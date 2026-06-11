package testhelpers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/pkg/testhelpers"
)

var _ = Describe("ResourceFromFile", func() {
	It("loads a ProjectDevelopmentStream sample", func() {
		var pds projctlv1beta1.ProjectDevelopmentStream
		testhelpers.ResourceFromFile("../../config/samples/projctl_v1beta1_pds_no_template.yaml", &pds)
		Expect(pds.Name).To(Equal("pds-no-template"))
		Expect(pds.Spec.Project).To(Equal("project-sample"))
	})
})
