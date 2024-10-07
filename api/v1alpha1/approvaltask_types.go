package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TaskApproved = "Approved"
	TaskRejected = "Rejected"
	TaskCanceled = "Canceled"
	TaskPending  = "Pending"

	ApprovalTaskKind = "ApprovalTask"
)

// ApprovalTaskSpec defines the desired state of ApprovalTask
type ApprovalTaskSpec struct {
	// Action is the action to be taken on the task.
	// +optional
	// +kubebuilder:validation:Enum=Pending;Approved;Rejected;Canceled
	// +kubebuilder:default=Pending
	Action string `json:"action,omitempty"`

	// Description that is shown to the user for the approval action.
	// +optional
	// +kubebuilder:default=Proceed
	Description string `json:"description,omitempty"`

	// Approve is the approval information.
	// +optional
	Approve *Approve `json:"approve,omitempty"`
}

type Approve struct {
	// ApprovedBy is indicating the identity of the approver.
	// +required
	ApprovedBy string `json:"approvedBy"`

	// Comment is the comment provided by the approver.
	// +optional
	Comment string `json:"comment,omitempty"`
}

type PipelineRef struct {
	// Name of the Tekton pipeline.
	// +required
	Name string `json:"name"`
}

// ApprovalTaskStatus defines the observed state of ApprovalTask
type ApprovalTaskStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ApprovalTask is the Schema for the approvaltasks API
type ApprovalTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApprovalTaskSpec   `json:"spec,omitempty"`
	Status ApprovalTaskStatus `json:"status,omitempty"`
}

func (a *ApprovalTask) IsApproved() bool {
	return a.Spec.Action == TaskApproved
}

func (a *ApprovalTask) IsRejected() bool {
	return a.Spec.Action == TaskRejected
}

//+kubebuilder:object:root=true

// ApprovalTaskList contains a list of ApprovalTask
type ApprovalTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApprovalTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApprovalTask{}, &ApprovalTaskList{})
}
