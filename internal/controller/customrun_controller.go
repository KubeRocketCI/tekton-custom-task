package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	customTasksApi "github.com/KubeRocketCI/tekton-custom-task/api/v1alpha1"
	tektonPipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcileCustomRun struct {
	client client.Client
}

func NewReconcileCustomRun(cl client.Client) *ReconcileCustomRun {
	return &ReconcileCustomRun{client: cl}
}

// +kubebuilder:rbac:groups=edp.epam.com,resources=approvaltasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=edp.epam.com,resources=approvaltasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=edp.epam.com,resources=approvaltasks/finalizers,verbs=update
// +kubebuilder:rbac:groups=tekton.dev,resources=customruns,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=tekton.dev,resources=customruns/status,verbs=get;update;patch
// Reconcile is responsible for reconciling CustomRun objects with ApprovalTask references.
func (r *ReconcileCustomRun) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling CustomRun")

	run := &tektonPipelineApi.CustomRun{}
	if err := r.client.Get(ctx, req.NamespacedName, run); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get CustomRun instance from k8s: %w", err)
	}

	if !hasApprovalTaskRef(run) {
		return reconcile.Result{}, nil
	}

	if run.IsDone() {
		log.Info("Run is finished, done reconciling")
		return reconcile.Result{}, nil
	}

	if run.IsCancelled() || run.HasTimedOut(clock.RealClock{}) {
		if err := r.cancelApprovalTask(ctx, run); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	if run.Status.StartTime == nil {
		if err := r.setRunningStatus(ctx, run); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	if err := r.processApprovalTask(ctx, run); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Finished reconciling CustomRun")

	return ctrl.Result{
		RequeueAfter: time.Second * 5,
	}, nil
}

func (r *ReconcileCustomRun) setRunningStatus(ctx context.Context, run *tektonPipelineApi.CustomRun) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting new CustomRun")

	now := metav1.Now()
	run.Status.StartTime = &now
	run.Status.MarkCustomRunRunning("Running", "Pipeline run is pending approval")

	if err := r.client.Status().Update(ctx, run); err != nil {
		return fmt.Errorf("failed to update CustomRun status: %w", err)
	}

	return nil
}

func (r *ReconcileCustomRun) setSucceededStatus(
	ctx context.Context,
	run *tektonPipelineApi.CustomRun,
	results []tektonPipelineApi.CustomRunResult,
) error {
	now := metav1.Now()
	run.Status.CompletionTime = &now
	run.Status.MarkCustomRunSucceeded("Approved", "Pipeline run was approved")

	run.Status.Results = append(
		run.Status.Results,
		results...,
	)

	if err := r.client.Status().Update(ctx, run); err != nil {
		return fmt.Errorf("failed to update CustomRun status: %w", err)
	}

	return nil
}

func (r *ReconcileCustomRun) setFailedStatus(
	ctx context.Context,
	run *tektonPipelineApi.CustomRun,
	reason, message string,
	results []tektonPipelineApi.CustomRunResult,
) error {
	run.Status.MarkCustomRunFailed(reason, message)

	run.Status.Results = append(
		run.Status.Results,
		results...,
	)

	if err := r.client.Status().Update(ctx, run); err != nil {
		return fmt.Errorf("failed to update CustomRun status: %w", err)
	}

	return nil
}

func (r *ReconcileCustomRun) processApprovalTask(ctx context.Context, run *tektonPipelineApi.CustomRun) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Processing ApprovalTask")

	task := &customTasksApi.ApprovalTask{}

	err := r.client.Get(ctx, client.ObjectKey{
		Namespace: run.Namespace,
		Name:      makeApprovalTaskName(run),
	}, task)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			if err = r.createApprovalTask(ctx, run); err != nil {
				return err
			}

			return nil
		}

		return fmt.Errorf("failed to get ApprovalTask: %w", err)
	}

	if task.IsApproved() {
		log.Info("ApprovalTask is approved")

		if err = r.setSucceededStatus(ctx, run, getResults(task)); err != nil {
			return err
		}

		return nil
	}

	if task.IsRejected() {
		log.Info("ApprovalTask is rejected")

		if err = r.setFailedStatus(ctx, run, "Rejected", "Pipeline run was rejected", getResults(task)); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (r *ReconcileCustomRun) createApprovalTask(ctx context.Context, run *tektonPipelineApi.CustomRun) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating ApprovalTask")

	task := &customTasksApi.ApprovalTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeApprovalTaskName(run),
			Namespace: run.Namespace,
			Labels:    getCustomRunLabels(run),
		},
		Spec: customTasksApi.ApprovalTaskSpec{
			Description: getApprovalTaskDescription(run),
		},
	}

	if err := controllerutil.SetControllerReference(run, task, r.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for ApprovalTask: %w", err)
	}

	if err := r.client.Create(ctx, task); err != nil {
		return fmt.Errorf("failed to create ApprovalTask: %w", err)
	}

	log.Info("ApprovalTask created")

	return nil
}

func (r *ReconcileCustomRun) cancelApprovalTask(ctx context.Context, run *tektonPipelineApi.CustomRun) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Cancelling ApprovalTask")

	task := &customTasksApi.ApprovalTask{}

	err := r.client.Get(ctx, client.ObjectKey{
		Namespace: run.Namespace,
		Name:      makeApprovalTaskName(run),
	}, task)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return fmt.Errorf("failed to get ApprovalTask for cancellation: %w", err)
	}

	if err == nil {
		task.Spec.Action = customTasksApi.TaskCanceled
		if err = r.client.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to cancel ApprovalTask: %w", err)
		}

		log.Info("ApprovalTask canceled")
	}

	if err = r.setFailedStatus(ctx, run, "Canceled", "Pipeline run was canceled", getResults(task)); err != nil {
		return err
	}

	log.Info("CustomRun canceled")

	return nil
}

func (r *ReconcileCustomRun) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tektonPipelineApi.CustomRun{}).
		Complete(r)
}

type CustomTaskRef struct {
	APIVersion string
	Kind       string
}

func getCustomTaskRef(customRun *tektonPipelineApi.CustomRun) *CustomTaskRef {
	if customRun.Spec.CustomSpec != nil {
		return &CustomTaskRef{
			APIVersion: customRun.Spec.CustomSpec.APIVersion,
			Kind:       customRun.Spec.CustomSpec.Kind,
		}
	}

	if customRun.Spec.CustomRef != nil {
		return &CustomTaskRef{
			APIVersion: customRun.Spec.CustomRef.APIVersion,
			Kind:       string(customRun.Spec.CustomRef.Kind),
		}
	}

	return nil
}

func hasApprovalTaskRef(run *tektonPipelineApi.CustomRun) bool {
	customTaskRef := getCustomTaskRef(run)

	if customTaskRef == nil ||
		customTaskRef.APIVersion != customTasksApi.GroupVersion.String() ||
		customTaskRef.Kind != customTasksApi.ApprovalTaskKind {
		return false
	}

	return true
}

func makeApprovalTaskName(run *tektonPipelineApi.CustomRun) string {
	return fmt.Sprintf("%s-approval", run.Name)
}

const (
	tektonPipelineLabel     = "tekton.dev/pipeline"
	tektonPipelineRunLabel  = "tekton.dev/pipelineRun"
	tektonPipelineTaskLabel = "tekton.dev/pipelineTask"
)

func getCustomRunLabels(run *tektonPipelineApi.CustomRun) map[string]string {
	l := make(map[string]string, len(run.Labels))

	if v, ok := run.Labels[tektonPipelineLabel]; ok {
		l[tektonPipelineLabel] = v
	}

	if v, ok := run.Labels[tektonPipelineRunLabel]; ok {
		l[tektonPipelineRunLabel] = v
	}

	if v, ok := run.Labels[tektonPipelineTaskLabel]; ok {
		l[tektonPipelineTaskLabel] = v
	}

	return l
}

const descriptionParamName = "description"

func getApprovalTaskDescription(run *tektonPipelineApi.CustomRun) string {
	d := run.Spec.GetParam(descriptionParamName)
	if d != nil {
		return d.Value.StringVal
	}

	return ""
}

func getResults(task *customTasksApi.ApprovalTask) []tektonPipelineApi.CustomRunResult {
	if task == nil {
		return []tektonPipelineApi.CustomRunResult{
			{
				Name:  "approved",
				Value: "false",
			},
		}
	}

	r := []tektonPipelineApi.CustomRunResult{
		{
			Name:  "approved",
			Value: strconv.FormatBool(task.IsApproved()),
		},
	}

	if task.Spec.Approve != nil {
		r = append(r,
			tektonPipelineApi.CustomRunResult{
				Name:  "approvedBy",
				Value: task.Spec.Approve.ApprovedBy,
			},
			tektonPipelineApi.CustomRunResult{
				Name:  "comment",
				Value: task.Spec.Approve.Comment,
			},
		)
	}

	return r
}
