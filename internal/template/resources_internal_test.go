package template

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
			var allAPIsWFinalizerAccessNeeded []apischema.GroupVersionKind
			for _, srt := range supportedResourceTypes {
				allSupportedAPIs = append(allSupportedAPIs, srt.supportedAPIs...)
				if srt.ownerDeletionBlocked || srt.ownerIsController {
					allAPIsWFinalizerAccessNeeded = append(allAPIsWFinalizerAccessNeeded, srt.ownerAPI)
				}
			}
			allSupportedAPIEntries := make([]TableEntry, 0, len(allSupportedAPIs))
			for _, gvk := range allSupportedAPIs {
				allSupportedAPIEntries = append(allSupportedAPIEntries, Entry(nil, gvk))
			}
			allAPIsWFinAccessNdedEntries := make([]TableEntry, 0, len(allAPIsWFinalizerAccessNeeded))
			for _, gvk := range allAPIsWFinalizerAccessNeeded {
				allAPIsWFinAccessNdedEntries = append(allAPIsWFinAccessNdedEntries, Entry(nil, gvk))
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
			DescribeTable(
				"For each API GVK we need finalizer update permissions on",
				func(api apischema.GroupVersionKind) {
					Expect(managerRole.Rules).To(ContainElement(
						MatchFields(IgnoreExtras, Fields{
							"APIGroups": ContainElement(api.Group),
							"Resources": ContainElement(fmt.Sprintf(
								"%s/finalizers", plz.Plural(strings.ToLower(api.Kind))),
							),
							"ResourceNames": BeEmpty(),
							"Verbs":         ContainElements("update"),
						}),
					))
				},
				allAPIsWFinAccessNdedEntries,
			)
		})
	})

	Describe("validateResourceNameFields", func() {
		var res *unstructured.Unstructured

		BeforeEach(func() {
			res = &unstructured.Unstructured{
				Object: map[string]any{
					"key1": map[string]any{
						"key1a": "good-name",
						"key1b": []any{
							"good-name1",
							"good-name2",
						},
					},
					"key2": map[string]any{
						"key2a": "bad.name",
						"key2b": []any{
							"good-name1",
							"bad.name2",
						},
					},
				},
			}
		})

		DescribeTable(
			"it ensures good k8s name values in specified field paths",
			func(nameFields [][]string) {
				Expect(validateResourceNameFields(res, nameFields)).To(Succeed())
			},
			Entry("checks string fields", [][]string{{"key1", "key1a"}}),
			Entry("checks slice-of-strings fields", [][]string{{"key1", "key1b", "[]"}}),
		)

		DescribeTable(
			"it finds bad k8s name values in specified field paths",
			func(nameFields [][]string) {
				Expect(validateResourceNameFields(res, nameFields)).ToNot(Succeed())
			},
			Entry("checks string fields", [][]string{
				{"key2", "key2a"},
			}),
			Entry("can check multiple string fields", [][]string{
				{"key1", "key1a"},
				{"key2", "key2a"},
			}),
			Entry("checks slice-of-strings fields", [][]string{
				{"key2", "key2b", "[]"},
			}),
			Entry("can check multiple fields of different types", [][]string{
				{"key1", "key1a"},
				{"key2", "key2b", "[]"},
				{"key2", "key2a"},
			}),
		)
	})
})
