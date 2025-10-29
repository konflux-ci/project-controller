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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	projctlv1beta1 "github.com/konflux-ci/project-controller/api/v1beta1"
	"github.com/konflux-ci/project-controller/internal/ownership"
	"github.com/konflux-ci/project-controller/internal/template"
	"github.com/konflux-ci/project-controller/pkg/logr/eventr"
	"github.com/konflux-ci/project-controller/pkg/logr/muxr"
)

const (
	// ConditionTypeReady represents the Ready condition type
	ConditionTypeReady = "Ready"
)

// ProjectDevelopmentStreamReconciler reconciles a ProjectDevelopmentStream object
type ProjectDevelopmentStreamReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreams/finalizers,verbs=update

//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projects,verbs=get;list;watch
//+kubebuilder:rbac:groups=projctl.konflux.dev,resources=projectdevelopmentstreamtemplates,verbs=get;list;watch

//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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
	log = muxr.NewMuxLogger(log, eventr.NewEventr(r.Recorder, &pds))
	// Update context with the enriched logger so that setReadyCondition can use it
	ctx = ctrl.LoggerInto(ctx, log)

	// Set initial condition
	_ = r.setReadyCondition(ctx, &pds, metav1.ConditionUnknown, "Reconciling", "Reconciling ProjectDevelopmentStream")

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
			// run can start with updated owner ref
			_ = r.setReadyCondition(ctx, &pds, metav1.ConditionUnknown, "UpdatingOwnerRef", "Owner reference updated, re-reconciling")
			return ctrl.Result{}, nil
		}
	}

	var templateName string
	if pds.Spec.Template == nil {
		log.Info("No template is associated with this ProjectDevelopmentStream")
		_ = r.setReadyCondition(ctx, &pds, metav1.ConditionTrue, "NoTemplate", "ProjectDevelopmentStream ready (no template specified)")
		return ctrl.Result{}, nil
	}
	templateName = pds.Spec.Template.Name
	log = log.WithValues("PDS Template", templateName)
	ctx = ctrl.LoggerInto(ctx, log)

	var pdst projctlv1beta1.ProjectDevelopmentStreamTemplate
	templateKey := client.ObjectKey{Namespace: pds.GetNamespace(), Name: templateName}
	if err := r.Get(ctx, templateKey, &pdst); err != nil {
		log.Error(err, "Failed to fetch template")
		_ = r.setReadyCondition(ctx, &pds, metav1.ConditionFalse, "TemplateFetchFailed", fmt.Sprintf("Failed to fetch template: %v", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info(fmt.Sprintf("Applying resources from ProjectDevelopmentStreamTemplate: %s", pdst.Name))
	resources, err := template.MkResources(pds, pdst)
	if err != nil {
		log.Error(err, "Failed to generate resources from template")
		_ = r.setReadyCondition(ctx, &pds, metav1.ConditionFalse, "TemplateGenerationFailed", fmt.Sprintf("Failed to generate resources from template: %v", err))
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
		log.V(1).Info("Creating/Updating resource")
		ownership.AddMissingUIDs(ctx, r.Client, resource)
		if len(resource.GetOwnerReferences()) <= 0 {
			// If the resource does not have an owner set, use the PDS
			_ = controllerutil.SetOwnerReference(&pds, resource, r.Scheme)
		}
		requeue = requeue || r.createOrUpdateResource(ctx, log, resource)
	}

	// Set final condition based on whether we need to requeue
	if requeue {
		_ = r.setReadyCondition(ctx, &pds, metav1.ConditionUnknown, "ApplyingResources", "Resource conflicts detected, retrying")
	} else {
		_ = r.setReadyCondition(ctx, &pds, metav1.ConditionTrue, "ResourcesApplied", "All resources applied successfully")
	}

	return ctrl.Result{Requeue: requeue}, nil
}

// Create or update the given resource. Returns true if there is an update
// conflict for the resource and therefore the reconcile action should be
// re-queued.
func (r *ProjectDevelopmentStreamReconciler) createOrUpdateResource(ctx context.Context, log logr.Logger, resource *unstructured.Unstructured) bool {
	err := r.Client.Patch(
		ctx,
		resource,
		client.Apply, //nolint:staticcheck // deprecated: will be migrated to new Apply API in future
		client.FieldOwner("projctl.konflux.dev"),
		client.ForceOwnership,
	)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to create or update resource: %s [%s]", resource.GetName(), resource.GetKind()))
		return apierrors.IsConflict(err)
	}
	log.Info(fmt.Sprintf("Resource updated: %s [%s]", resource.GetName(), resource.GetKind()))
	return false
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

// Returns a handler for collecting all dev streams that exist on the same namespace as
// the object passed to the handler
func getSameNSEventHandler(r *ProjectDevelopmentStreamReconciler) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(ctx context.Context, o client.Object) []reconcile.Request {
			lg := log.FromContext(ctx)

			// get all streams from current namespace
			list := projctlv1beta1.ProjectDevelopmentStreamList{}
			if err := r.Client.List(ctx, &list, client.InNamespace(o.GetNamespace())); err != nil {
				lg.Error(err, "Failed listing dev streams in namespace")
				return nil
			}
			ret := make([]reconcile.Request, len(list.Items))

			for i := range list.Items {
				ret[i] = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])}
			}
			return ret
		},
	)
}

// setReadyCondition sets the Ready condition and updates the status
func (r *ProjectDevelopmentStreamReconciler) setReadyCondition(ctx context.Context, pds *projctlv1beta1.ProjectDevelopmentStream, status metav1.ConditionStatus, reason, message string) error {
	log := log.FromContext(ctx)

	condition := metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             status,
		ObservedGeneration: pds.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing Ready condition
	for i, existing := range pds.Status.Conditions {
		if existing.Type == ConditionTypeReady {
			// Preserve LastTransitionTime when status hasn't changed per Kubernetes API conventions.
			// LastTransitionTime should only update when the condition's status field transitions
			// (True/False/Unknown), not when reason or message changes.
			if existing.Status == status {
				condition.LastTransitionTime = existing.LastTransitionTime
			}
			pds.Status.Conditions[i] = condition
			if err := r.Status().Update(ctx, pds); err != nil {
				log.Error(err, "Failed to update Ready condition", "reason", reason)
				return err
			}
			return nil
		}
	}

	// Condition not found, append new one
	pds.Status.Conditions = append(pds.Status.Conditions, condition)
	if err := r.Status().Update(ctx, pds); err != nil {
		log.Error(err, "Failed to update Ready condition", "reason", reason)
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectDevelopmentStreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&projctlv1beta1.ProjectDevelopmentStream{}).
		Watches(
			&projctlv1beta1.ProjectDevelopmentStreamTemplate{},
			getSameNSEventHandler(r),
		).
		Watches(
			&projctlv1beta1.Project{},
			getSameNSEventHandler(r),
		).
		Complete(r)
}
