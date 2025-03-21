package controller

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// JobCleanupReconciler cleans up completed/failed jobs
type JobCleanupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;delete
func (r *JobCleanupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	job := &batchv1.Job{}
	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to fetch Job")
		return ctrl.Result{}, err
	}
	if _, exists := job.Labels["backup-database"]; !exists {
		log.Info("Skipping cleanup for non-backup job", "job", job.Name)
		return ctrl.Result{}, nil
	}
	if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
		log.Info("Cleaning up completed/failed job", "job", job.Name)

		if err := r.Delete(ctx, job); err != nil {
			log.Error(err, "Failed to delete job")
			return ctrl.Result{}, err
		}
	}
	// Requeue periodically (e.g., every 1 minute)
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *JobCleanupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Named("jobcleanup").
		Complete(r)
}
