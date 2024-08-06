package template

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var someValues = map[string]string{
	"foo": "bar",
	"baz": "bal",
}

var _ = DescribeTable(
	"applyFieldTemplate aplies a template in a field within a nested structure",
	func(obj map[string]any, path []string, values map[string]string, expected map[string]any) {
		err := applyFieldTemplate(obj, path, values)

		Expect(err).NotTo(HaveOccurred())
		Expect(obj).To(Equal(expected))
	},
	Entry(
		"where path defines which key to apply to",
		map[string]any{
			"key1": "{{.foo}}",
			"key2": "{{.baz}}",
		},
		[]string{"key1"},
		someValues,
		map[string]any{
			"key1": "bar",
			"key2": "{{.baz}}",
		},
	),
	Entry(
		"where multiple path values define key recursion",
		map[string]any{
			"key1": map[string]any{
				"key1a": "{{.foo}}",
				"key1b": "{{.baz}}",
			},
			"key2": map[string]any{
				"key2a": "{{.baz}}",
			},
		},
		[]string{"key1", "key1b"},
		someValues,
		map[string]any{
			"key1": map[string]any{
				"key1a": "{{.foo}}",
				"key1b": "bal",
			},
			"key2": map[string]any{
				"key2a": "{{.baz}}",
			},
		},
	),
	Entry(
		"and not-found pathes are ignored",
		map[string]any{
			"key1": "{{.foo}}",
			"key2": "{{.baz}}",
		},
		[]string{"key3", "key3a"},
		someValues,
		map[string]any{
			"key1": "{{.foo}}",
			"key2": "{{.baz}}",
		},
	),
	Entry(
		"and pathes that end with [] point to lists, all members are applied",
		map[string]any{
			"key1": []any{
				"{{.foo}}",
				"{{.baz}}",
			},
		},
		[]string{"key1", "[]"},
		someValues,
		map[string]any{
			"key1": []any{
				"bar",
				"bal",
			},
		},
	),
)
