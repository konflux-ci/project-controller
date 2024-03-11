/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/internal/template"
)

// ProjectDevelopmentStreamReconciler reconciles a ProjectDevelopmentStream object
type ProjectDevelopmentStreamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProjectDevelopmentStream object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *ProjectDevelopmentStreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	log := log.FromContext(ctx)

	var pds projctlv1beta1.ProjectDevelopmentStream
	if err := r.Get(ctx, req.NamespacedName, &pds); err != nil {
		log.Error(err, "Unable to fetch ProjectDevelopmentStream")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log = log.WithValues("PDS name", pds.ObjectMeta.Name)

	// This is arguably better done in an admission hook, but its easier to test
	// when doing this from the controller
	if !r.checkProductOwnerRef(pds) {
		log.Info("Setting ownerReference for ProductDevelopmentStream")
		if err := r.setProductOwnerRef(ctx, &pds); err != nil {
			log.Error(err, "Error setting product ownerReference for ProjectDevelopmentStream")
			// We treat the product association as a light requirement so we
			// continue to applying templates rather then quitting on error here
		} else {
			// Since we modified the PDS object exit so another reconciliation
			// run can start
			return ctrl.Result{}, nil
		}
	}

	var templateName string
	if pds.Spec.Template == nil {
		log.Info("No template is associated with this ProjectDevelopmentStream")
		return ctrl.Result{}, nil
	}
	templateName = pds.Spec.Template.Name
	log = log.WithValues("PDS Template", templateName)

	var pdst projctlv1beta1.ProjectDevelopmentStreamTemplate
	templateKey := client.ObjectKey{Namespace: pds.GetNamespace(), Name: templateName}
	if err := r.Get(ctx, templateKey, &pdst); err != nil {
		log.Error(err, "Failed to fetch template")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Applying resources from ProjectDevelopmentStreamTemplate")
	resources, err := template.MkResources(pds, pdst)
	if err != nil {
		log.Error(err, "Failed to generate resources from template")
		// We return 'nil' error because there is not point retrying the
		// reconcile loop
		return ctrl.Result{}, nil
	}

	var requeue bool
	for _, resource := range resources {
		log := log.WithValues(
			"apiVersion", resource.GetAPIVersion(),
			"kind", resource.GetKind(),
			"name", resource.GetName(),
		)
		log.Info("Creating/Updating resource")

		requeue = requeue || r.createOrUpdateResource(ctx, log, resource)
	}

	return ctrl.Result{Requeue: requeue}, nil
}

func (r *ProjectDevelopmentStreamReconciler) createOrUpdateResource(ctx context.Context, log logr.Logger, resource *unstructured.Unstructured) (isUpdateConflict bool) {
	var existing unstructured.Unstructured
	existing.SetAPIVersion(resource.GetAPIVersion())
	existing.SetKind(resource.GetKind())
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(resource), &existing); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Creating new resource")
			if err := r.Client.Create(ctx, resource); err != nil {
				log.Error(err, "Failed to create resource")
			}
		} else {
			log.Error(err, "Failed to read existing resource")
		}
		return
	}
	update := existing.DeepCopy()
	if m, ok, _ := unstructured.NestedMap(resource.Object, "spec"); ok {
		if err := unstructured.SetNestedMap(update.Object, m, "spec"); err != nil {
			log.Error(err, "Failed to update 'spec' for generated resource")
		}
	}
	if equality.Semantic.DeepEqual(existing.Object, update.Object) {
		log.Info("Resource already up to date")
		return
	}
	if err := r.Client.Update(ctx, update); err != nil {
		log.Error(err, "Failed to update resource")
		isUpdateConflict = apierrors.IsConflict(err)
		return
	}
	log.Info("Resource updated")
	return
}

// Check wither the PDS ownerReference is already set to point to the right
// product
func (r *ProjectDevelopmentStreamReconciler) checkProductOwnerRef(pds projctlv1beta1.ProjectDevelopmentStream) bool {
	projectName := pds.Spec.Project
	if projectName == "" {
		return true // We define an empty project field as having a reference
	}
	projectGVK, _ := r.Client.GroupVersionKindFor(&projctlv1beta1.Project{})
	prjAPIVersion, prjKind := projectGVK.ToAPIVersionAndKind()
	for _, ref := range pds.ObjectMeta.OwnerReferences {
		if ref.APIVersion == prjAPIVersion && ref.Kind == prjKind && ref.Name == projectName {
			return true
		}
	}
	return false
}

func (r *ProjectDevelopmentStreamReconciler) setProductOwnerRef(ctx context.Context, pds *projctlv1beta1.ProjectDevelopmentStream) error {
	projectKey := client.ObjectKey{Namespace: pds.GetNamespace(), Name: pds.Spec.Project}
	project := projctlv1beta1.Project{}
	if err := r.Client.Get(ctx, projectKey, &project); err != nil {
		return err
	}
	if err := controllerutil.SetOwnerReference(&project, pds, r.Scheme); err != nil {
		return err
	}
	return r.Client.Update(ctx, pds)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectDevelopmentStreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&projctlv1beta1.ProjectDevelopmentStream{}).
		Complete(r)
}
