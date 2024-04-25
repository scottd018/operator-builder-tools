// SPDX-License-Identifier: MIT

package phases

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nukleros/operator-builder-tools/pkg/controller/workload"
	"github.com/nukleros/operator-builder-tools/pkg/resources"
)

const (
	WorkloadFinalizerName = "Finalizer"
	ChildFinalizerName    = "ChildFinalizer"
)

// DeletePhase is the phase to delete all of the resources and ensure they do not exist.
func DeletePhase(r workload.Reconciler, req *workload.Request, options ...ResourceOption) (bool, error) {
	// reverse the order of the delete so that they are deleted in the reverse order
	// that they are deployed
	delete, err := r.GetResources(req)
	if err != nil {
		return false, fmt.Errorf("unable to get resources - %w", err)
	}

	for i, j := 0, len(delete)-1; i < j; i, j = i+1, j-1 {
		delete[i], delete[j] = delete[j], delete[i]
	}

	// loop through the reversed resources and delete the resource and ensure
	// they do not exist
	for _, resource := range delete {
		current, err := resources.Get(r, req, resource)
		if err != nil {
			// continue the loop if the resource is already gone
			if errors.IsNotFound(err) {
				continue
			}

			return false, err
		}

		if current == nil {
			continue
		}

		if containsString(current.GetFinalizers(), finalizerName(req, ChildFinalizerName)) {
			original, ok := current.DeepCopyObject().(client.Object)
			if !ok {
				return false, fmt.Errorf("unable to convert child resource to client.Object, %w", err)
			}

			controllerutil.RemoveFinalizer(current, finalizerName(req, ChildFinalizerName))

			r.GetLogger().Info(
				"deleting resource",
				"kind", current.GetObjectKind().GroupVersionKind().Kind,
				"name", current.GetName(),
				"namespace", current.GetNamespace(),
			)

			if err := r.Patch(req.Context, current, client.MergeFrom(original)); err != nil {
				return false, fmt.Errorf("unable to remove finalizer - %w", err)
			}
		}

		// re-attempt to get the resource to ensure it does not exist
		err = resources.Delete(r, req, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return false, err
		}

		// re-attempt to get the resource to ensure it does not exist
		current, err = resources.Get(r, req, resource)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return false, err
		}

		// if we found the resource, do not continue on with the phase but return no error
		if current != nil {
			return false, nil
		}
	}

	return true, nil
}

// DeletionCompletePhase executes the completion of a reconciliation loop for a delete request.
func DeletionCompletePhase(r workload.Reconciler, req *workload.Request, options ...ResourceOption) (bool, error) {
	req.Log.Info("successfully deleted")

	return true, nil
}

// RegisterDeleteHooks adds finializers to the workload resources so that the delete lifecycle can be run beofre the object is deleted.
func RegisterDeleteHooks(r workload.Reconciler, req *workload.Request) error {
	workloadFinalizer := finalizerName(req, WorkloadFinalizerName)

	if req.Workload.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(req.Workload.GetFinalizers(), workloadFinalizer) {
			controllerutil.AddFinalizer(req.Workload, workloadFinalizer)

			if err := r.Update(req.Context, req.Workload); err != nil {
				return fmt.Errorf("unable to register delete hook on %s, %w", req.Workload.GetWorkloadGVK().Kind, err)
			}
		}
	}

	return nil
}

// finalizerName returns the finalizer name given a suffix.
func finalizerName(req *workload.Request, suffix string) string {
	return fmt.Sprintf("%s/%s", req.Workload.GetWorkloadGVK().Group, suffix)
}

// containsString checks for a string in a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}
