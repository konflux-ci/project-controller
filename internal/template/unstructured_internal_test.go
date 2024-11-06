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
	"applyFieldTemplate applies a template in a field within a nested structure",
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
		"and paths that end with [] point to lists, all members are applied",
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
	Entry(
		"and paths that contain [] point to lists of non-scalar values",
		map[string]any{
			"key1": []any{
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": "{{.baz}}",
				},
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": "{{.baz}}",
				},
				map[string]any{
					"key1c": "{{.foo}}",
					"key1d": "{{.baz}}",
				},
			},
		},
		[]string{"key1", "[]", "key1b"},
		someValues,
		map[string]any{
			"key1": []any{
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": "bal",
				},
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": "bal",
				},
				map[string]any{
					"key1c": "{{.foo}}",
					"key1d": "{{.baz}}",
				},
			},
		},
	),
	Entry(
		"and paths may contain [] multiple times",
		map[string]any{
			"key1": []any{
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": []any{
						map[string]any{
							"key1a":  "{{.foo}}",
							"key1b1": "{{.baz}}",
						},
					},
				},
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": "{{.baz}}",
				},
				map[string]any{
					"key1c": "{{.foo}}",
					"key1d": "{{.baz}}",
				},
			},
		},
		[]string{"key1", "[]", "key1b", "[]", "key1b1"},
		someValues,
		map[string]any{
			"key1": []any{
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": []any{
						map[string]any{
							"key1a":  "{{.foo}}",
							"key1b1": "bal",
						},
					},
				},
				map[string]any{
					"key1a": "{{.foo}}",
					"key1b": "{{.baz}}",
				},
				map[string]any{
					"key1c": "{{.foo}}",
					"key1d": "{{.baz}}",
				},
			},
		},
	),
	Entry(
		"and paths that end with [] can be nested inside other lists",
		map[string]any{
			"key1": []any{
				map[string]any{
					"key1a": []any{
						"{{.foo}}",
						"{{.baz}}",
					},
				},
			},
			"key2": "{{.baz}}",
		},
		[]string{"key1", "[]", "key1a", "[]"},
		someValues,
		map[string]any{
			"key1": []any{
				map[string]any{
					"key1a": []any{
						"bar",
						"bal",
					},
				},
			},
			"key2": "{{.baz}}",
		},
	),
)
