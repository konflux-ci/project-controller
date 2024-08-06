package template

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable(
	"Execute applies a template string",
	func(template string, values map[string]string, expected string) {
		out, err := executeTemplate(template, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(Equal(expected))
	},
	Entry(
		"for simple template and value",
		"{{.version}}",
		map[string]string{"version": "1.2.3"},
		"1.2.3",
	),
	Entry(
		"and suports the hyphenize function",
		"{{.version|hyphenize}}",
		map[string]string{"version": "1.2.3"},
		"1-2-3",
	),
)
