// SPDX-License-Identifier: MIT

package phases

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nukleros/operator-builder-tools/pkg/controller/workload"
	"github.com/nukleros/operator-builder-tools/pkg/resources"
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

	skipCRDDeletion := hasResourceOption(ResourceOptionSkipDeleteCRD, options...)

	// loop through the reversed resources and delete the resource and ensure
	// they do not exist
	for _, resource := range delete {
		if skipCRDDeletion && resource.GetObjectKind().GroupVersionKind().Kind == resources.CustomResourceDefinitionKind {
			r.GetLogger().Info("skipping deletion of resource",
				"kind", resource.GetObjectKind().GroupVersionKind().Kind,
			)

			// get the resource from the cluster
			current, err := resources.Get(r, req, resource)
			if err != nil {
				return false, err
			}

			// remove owner references
			current.SetOwnerReferences(nil)

			// update the resource
			if err := r.Update(req.Context, current); err != nil {
				return false, err
			}

			continue
		}

		if err := resources.Delete(r, req, resource); err != nil {
			return false, err
		}

		// re-attempt to get the resource to ensure it does not exist
		if _, err := resources.Get(r, req, resource); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return false, err
		}
	}

	return true, nil
}

// DeletionCompletePhase executes the completion of a reconciliation loop for a delete request.
func DeletionCompletePhase(r workload.Reconciler, req *workload.Request, options ...ResourceOption) (bool, error) {
	req.Log.Info("successfully deleted")

	return true, nil
}

// RegisterDeleteHooks add finializers to the workload resources so that the delete lifecycle can be run beofre the object is deleted.
func RegisterDeleteHooks(r workload.Reconciler, req *workload.Request) error {
	myFinalizerName := fmt.Sprintf("%s/Finalizer", req.Workload.GetWorkloadGVK().Group)

	if req.Workload.GetDeletionTimestamp().IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(req.Workload.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(req.Workload, myFinalizerName)

			if err := r.Update(req.Context, req.Workload); err != nil {
				return fmt.Errorf("unable to register delete hook on %s, %w", req.Workload.GetWorkloadGVK().Kind, err)
			}
		}
	}

	return nil
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
