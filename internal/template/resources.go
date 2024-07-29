package template

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"

	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/internal/ownership"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apischema "k8s.io/apimachinery/pkg/runtime/schema"
)

//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=applications/finalizers,verbs=update
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=components,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=imagerepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=integrationtestscenarios,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=releaseplans,verbs=get;list;watch;create;update;patch;delete

// List of resource types supported by templates and various details about how
// to instantiate resources of those types. The list order determines the order
// in which resources are created, which can be significant for e.g. creating
// ownership relationships
var supportedResourceTypes = []struct {
	// The supported API group/version/kind values for this resource.
	supportedAPIs []apischema.GroupVersionKind
	// The list of template-able fields for the resource. Each member is a list
	// of strings indicating the full path to the field
	templateAbleFields [][]string
	// Like templateAbleFields but for fields that contain k8s resource names.
	// For such fields an error will be reported if the generated value does not
	// match ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	templateAbleNameFields [][]string
	// A field of the resource that points to its owner object. If nil, an
	// owner record would not be set.
	ownerNameField []string
	// The owner object GVK. Will only be used if ownerNameField is not empty
	ownerAPI apischema.GroupVersionKind
	// The owner object should be marked as the controller
	ownerIsController bool
	// The owner object deletion should be blocked
	ownerDeletionBlocked bool
}{
	{
		supportedAPIs: []apischema.GroupVersionKind{
			{Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "Application"},
		},
		templateAbleNameFields: [][]string{
			{"metadata", "name"},
		},
		templateAbleFields: [][]string{
			{"spec", "displayName"},
		},
	},
	{
		supportedAPIs: []apischema.GroupVersionKind{
			{Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "Component"},
		},
		templateAbleNameFields: [][]string{
			{"metadata", "name"},
			{"spec", "application"},
			{"spec", "componentName"},
		},
		templateAbleFields: [][]string{
			{"spec", "source", "git", "context"},
			{"spec", "source", "git", "dockerfileUrl"},
			{"spec", "source", "git", "revision"},
			{"spec", "source", "git", "url"},
		},
		ownerNameField: []string{"spec", "application"},
		ownerAPI: apischema.GroupVersionKind{
			Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "Application",
		},
	},
	{
		supportedAPIs: []apischema.GroupVersionKind{
			{Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "ImageRepository"},
		},
		templateAbleNameFields: [][]string{
			{"metadata", "name"},
			{"metadata", "labels", "appstudio.redhat.com/component"},
			{"metadata", "labels", "appstudio.redhat.com/application"},
			{"spec", "image", "name"},
		},
		ownerNameField: []string{"metadata", "labels", "appstudio.redhat.com/component"},
		ownerAPI: apischema.GroupVersionKind{
			Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "Component",
		},
	},
	{
		supportedAPIs: []apischema.GroupVersionKind{
			{Group: "appstudio.redhat.com", Version: "v1beta2", Kind: "IntegrationTestScenario"},
		},
		templateAbleNameFields: [][]string{
			{"metadata", "name"},
			{"spec", "application"},
			// TODO: Somehow allow templating spec.params and spec.resolverRef.params
			// which are arrays of name/value pairs. This would require changes to
			// applyResourceTemplate and possibly validateResourceNameFields
		},
		ownerNameField: []string{"spec", "application"},
		ownerAPI: apischema.GroupVersionKind{
			Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "Application",
		},
		ownerIsController:    true,
		ownerDeletionBlocked: true,
	},
	{
		supportedAPIs: []apischema.GroupVersionKind{
			{Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "ReleasePlan"},
		},
		templateAbleNameFields: [][]string{
			{"metadata", "name"},
			{"spec", "application"},
		},
		ownerNameField: []string{"spec", "application"},
		ownerAPI: apischema.GroupVersionKind{
			Group: "appstudio.redhat.com", Version: "v1alpha1", Kind: "Application",
		},
		ownerIsController:    true,
		ownerDeletionBlocked: true,
	},
}

// Make the resources to be owned by the given ProjectDevelopmentStream as
// defined by the given  ProjectDevelopmentStreamTemplate
func MkResources(
	pds projctlv1beta1.ProjectDevelopmentStream,
	pdst projctlv1beta1.ProjectDevelopmentStreamTemplate,
) ([]*unstructured.Unstructured, error) {
	resources := make([]*unstructured.Unstructured, 0, len(pdst.Spec.Resources))
	// unhandledTemplates is used to detect unsupported resource types that may
	// have been included in the template
	unhandledTemplates := make(map[int]bool, len(pdst.Spec.Resources))
	for i := range pdst.Spec.Resources {
		unhandledTemplates[i] = true
	}
	templateVarValues, err := getVarValues(pdst.Spec.Variables, pds.Spec.Template.Values)
	if err != nil {
		return nil, err
	}
	for _, srt := range supportedResourceTypes {
		for i, unstructuredObj := range pdst.Spec.Resources {
			if !findGVK(srt.supportedAPIs, unstructuredObj.GroupVersionKind()) {
				continue
			}
			unhandledTemplates[i] = false
			resource := unstructuredObj.Unstructured.DeepCopy()
			resource.SetNamespace(pds.GetNamespace())
			if err := applyResourceTemplate(resource, srt.templateAbleNameFields, templateVarValues); err != nil {
				return nil, err
			}
			if err := validateResourceNameFields(resource, srt.templateAbleNameFields); err != nil {
				return nil, err
			}
			if err := applyResourceTemplate(resource, srt.templateAbleFields, templateVarValues); err != nil {
				return nil, err
			}
			if srt.ownerNameField != nil {
				ownerName, ok, err := unstructured.NestedString(resource.Object, srt.ownerNameField...)
				if ok && err == nil {
					// If we can't find the owner name field, we just skip
					// setting an owner
					ownership.SetWithoutUid(
						resource,
						srt.ownerAPI,
						ownerName,
						srt.ownerIsController,
						srt.ownerDeletionBlocked,
					)
				}
			}
			resources = append(resources, resource)
		}
	}
	for i, unstructuredObj := range pdst.Spec.Resources {
		if unhandledTemplates[i] {
			return nil, fmt.Errorf(
				"Unsupported resource type in template: %s",
				unstructuredObj.GroupVersionKind(),
			)
		}
	}
	return resources, nil
}

func findGVK(GVKs []apischema.GroupVersionKind, someGVK apischema.GroupVersionKind) bool {
	for _, aGVK := range GVKs {
		if someGVK == aGVK {
			return true
		}
	}
	return false
}

// Given a resource, a list of template-able fields and template variable values,
// treat the fields as text/template templates and execute them generating new
// values for said fields
func applyResourceTemplate(
	resource *unstructured.Unstructured,
	templateAbleFields [][]string,
	templateVarValues map[string]string,
) error {
	for _, path := range templateAbleFields {
		valueTemplate, ok, err := unstructured.NestedString(resource.Object, path...)
		if err != nil {
			return fmt.Errorf("Error reading resource template: %s", err)
		}
		if !ok {
			continue
		}
		value, err := executeTemplate(valueTemplate, templateVarValues)
		if err != nil {
			return fmt.Errorf("Error applying resource template: %s", err)
		}
		err = unstructured.SetNestedField(resource.Object, value, path...)
		if err != nil {
			return fmt.Errorf("Error applying resource template: %s", err)
		}
	}
	return nil
}

var nameFieldPattern = regexp.MustCompile("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$")

// Given a resource and a list of field paths, check that the value in those
// paths conform to the k8s resource name rules and return a non-nil error if
// then don't.
func validateResourceNameFields(
	resource *unstructured.Unstructured,
	nameFields [][]string,
) error {
	for _, path := range nameFields {
		value, ok, err := unstructured.NestedString(resource.Object, path...)
		if err != nil || !ok {
			// We just ignore field reading errors, we'll deal with them elsewhere
			continue
		}
		if !nameFieldPattern.MatchString(value) {
			return fmt.Errorf(
				"Invalid resource name value '%s' for resource field '%s'. "+
					"Consider using the 'hyphenize' template function",
				value,
				strings.Join(path, "."),
			)
		}
	}
	return nil
}

// Get the values for the given template variables using the given values or
// the defaults if values ar missing
func getVarValues(
	vars []projctlv1beta1.ProjectDevelopmentStreamTemplateVariable,
	vals []projctlv1beta1.ProjectDevelopmentStreamSpecTemplateValue,
) (values map[string]string, err error) {
	values = map[string]string{}
	givenValues := map[string]string{}
	for _, val := range vals {
		givenValues[val.Name] = val.Value
	}
	for _, variable := range vars {
		if givenValue, ok := givenValues[variable.Name]; ok {
			values[variable.Name] = givenValue
		} else if variable.DefaultValue != nil {
			var value string
			if value, err = executeTemplate(*variable.DefaultValue, values); err != nil {
				break
			}
			values[variable.Name] = value
		} else {
			err = fmt.Errorf(
				"Template variable '%s' is missing a value and default not defined",
				variable.Name,
			)
			break
		}
	}
	return
}

var nameFieldInvalidCharPattern = regexp.MustCompile("[^a-z0-9]")
var templateFuncs = template.FuncMap{
	"hyphenize": func(str string) string {
		return nameFieldInvalidCharPattern.ReplaceAllString(str, "-")
	},
}

// Execute the template given as a string and return the result as a string
func executeTemplate(templateStr string, values map[string]string) (string, error) {
	theTemplate, err := template.New("").Funcs(templateFuncs).Parse(templateStr)
	if err != nil {
		return "", err
	}
	var valueBuf strings.Builder
	if err := theTemplate.Execute(&valueBuf, values); err != nil {
		return "", err
	}
	return valueBuf.String(), nil
}
