package template

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Given a possibly nested map structure, navigate to a particular scalar value
// using path - a list of string keys. Then treat that value as a template and
// apply it in-place while using the provided values.
func applyFieldTemplate(obj map[string]any, path []string, values map[string]string) error {
	valueTemplate, ok, err := unstructured.NestedString(obj, path...)
	if err != nil {
		return fmt.Errorf("error reading object template: %s", err)
	}
	if !ok {
		// If the path is not found in the structure, we ignore it
		return nil
	}
	value, err := executeTemplate(valueTemplate, values)
	if err != nil {
		return fmt.Errorf("error applying template: %s", err)
	}
	err = unstructured.SetNestedField(obj, value, path...)
	if err != nil {
		return fmt.Errorf("error applying template: %s", err)
	}
	return nil
}
