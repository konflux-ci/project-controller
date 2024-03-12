package ownership

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Given an object, find ownership records that are defined on it with missing
// UID values and fill-in those values.
// Access to an API client is and context needed to search for the owning
// objects.
func AddMissingUIDs(ctx context.Context, cli client.Client, object metav1.Object) {
	owners := object.GetOwnerReferences()
	for i, owner := range owners {
		if owner.UID != "" {
			continue
		}
		uid, err := findObjectUid(ctx, cli, owner.APIVersion, owner.Kind, object.GetNamespace(), owner.Name)
		if err != nil {
			// If we have an error fetching the owner object we just ignore it
			// and leave the record as-is. We rely on the fact that the server
			// will just strip it away if its still missing a UID
			continue
		}
		owners[i].UID = uid
	}
	object.SetOwnerReferences(owners)
}

func findObjectUid(ctx context.Context, cli client.Client, apiVersion, kind, namespace, name string) (types.UID, error) {
	key := client.ObjectKey{Namespace: namespace, Name: name}
	var object unstructured.Unstructured
	object.SetAPIVersion(apiVersion)
	object.SetKind(kind)
	if err := cli.Get(ctx, key, &object); err != nil {
		return "", err
	}
	return object.GetUID(), nil
}
