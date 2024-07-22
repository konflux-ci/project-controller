package template

import (
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	apischema "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gertd/go-pluralize"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/konflux-ci/project-controller/pkg/testhelpers"
)

var _ = Describe("Resources", func() {
	Describe("supportedResourceTypes lists resources supported in templates", func() {
		Describe("Each supported resource type should also have a matching RBAC rule", Ordered, func() {
			var managerRole rbacv1.ClusterRole
			var plz *pluralize.Client

			var allSupportedAPIs []apischema.GroupVersionKind
			for _, srt := range supportedResourceTypes {
				allSupportedAPIs = append(allSupportedAPIs, srt.supportedAPIs...)
			}
			allSupportedAPIEntries := make([]TableEntry, 0, len(allSupportedAPIs))
			for _, gvk := range allSupportedAPIs {
				allSupportedAPIEntries = append(allSupportedAPIEntries, Entry(nil, gvk))
			}

			BeforeAll(func() {
				plz = pluralize.NewClient()
			})

			BeforeAll(func() {
				testhelpers.ResourceFromFile("../../config/rbac/role.yaml", &managerRole)
				Expect(managerRole.Rules).ShouldNot(BeEmpty())
			})

			DescribeTable(
				"For each supported API GVK",
				func(supportedAPI apischema.GroupVersionKind) {
					Expect(managerRole.Rules).To(ContainElement(
						MatchFields(IgnoreExtras, Fields{
							"APIGroups":     ContainElement(supportedAPI.Group),
							"Resources":     ContainElement(plz.Plural(strings.ToLower(supportedAPI.Kind))),
							"ResourceNames": BeEmpty(),
							"Verbs": ContainElements(
								"create",
								"delete",
								"get",
								"list",
								"patch",
								"update",
								"watch",
							),
						}),
					))
				},
				allSupportedAPIEntries,
			)
		})
	})
})
