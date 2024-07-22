package ownership

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Set an ownership record for a resource without UID. This is meant to be used
// locally when the UID for the owning resource is not yet known.
// This function also ensure each object only has one ownership record for
// a particular API group and Kind combination.
func SetWithoutUid(
	resource metav1.Object, ownerAPI schema.GroupVersionKind, ownerName string,
	ownerIsControlller, ownerDeletionBlocked bool,
) {
	apiVersion, kind := ownerAPI.ToAPIVersionAndKind()
	ref := metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       ownerName,
	}
	if ownerIsControlller {
		ref.Controller = &ownerIsControlller
	}
	if ownerDeletionBlocked {
		ref.BlockOwnerDeletion = &ownerDeletionBlocked
	}
	upsertOwnerRef(ref, resource)
}

// Insert or update and ownership reference for the given object. A reference
// is updated if it refers to the same API group and kind as the given reference
func upsertOwnerRef(ref metav1.OwnerReference, object metav1.Object) {
	owners := object.GetOwnerReferences()
	if idx := indexOwnerRef(owners, ref); idx == -1 {
		owners = append(owners, ref)
	} else {
		owners[idx] = ref
	}
	object.SetOwnerReferences(owners)
}

// Find, within the given slice an ownerReference record that refers to the
// same Group and Kind as the given reference
func indexOwnerRef(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) int {
	for i, r := range ownerReferences {
		if referSameGroupKind(r, ref) {
			return i
		}
	}
	return -1
}

// Returns true if both given references refer to the same resource Group and
// Kind
func referSameGroupKind(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}
	return aGV.Group == bGV.Group && a.Kind == b.Kind
}
