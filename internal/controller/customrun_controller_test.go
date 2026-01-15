package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	customTasksApi "github.com/KubeRocketCI/tekton-custom-task/api/v1alpha1"
	tektonPipelineApi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("CodebaseIntegration controller", func() {
	It("Should create ApprovalTask", func() {
		By("Creating CustomRun object")
		run := &tektonPipelineApi.CustomRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-custom-run1",
				Namespace: ns,
				Labels: map[string]string{
					tektonPipelineLabel:     "test-pipeline",
					tektonPipelineRunLabel:  "test-pipeline-run",
					tektonPipelineTaskLabel: "test-task",
				},
			},
			Spec: tektonPipelineApi.CustomRunSpec{
				CustomRef: &tektonPipelineApi.TaskRef{
					APIVersion: customTasksApi.GroupVersion.String(),
					Kind:       customTasksApi.ApprovalTaskKind,
				},
				Params: []tektonPipelineApi.Param{
					{
						Name: descriptionParamName,
						Value: tektonPipelineApi.ParamValue{
							Type:      tektonPipelineApi.ParamTypeString,
							StringVal: "Test description",
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, run)).Should(Succeed())
		By("Checking ApprovalTask creation")
		Eventually(func(g Gomega) {
			task := &customTasksApi.ApprovalTask{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: makeApprovalTaskName(run), Namespace: ns}, task)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(task.Spec.Description).Should(Equal("Test description"))
			g.Expect(task.GetLabels()).Should(Equal(getCustomRunLabels(run)))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	})
	testApprovalTaskAction := func(runName, action, comment, approvedValue string, shouldSucceed bool) {
		By("Creating CustomRun object")
		run := &tektonPipelineApi.CustomRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runName,
				Namespace: ns,
			},
			Spec: tektonPipelineApi.CustomRunSpec{
				CustomRef: &tektonPipelineApi.TaskRef{
					APIVersion: customTasksApi.GroupVersion.String(),
					Kind:       customTasksApi.ApprovalTaskKind,
				},
			},
		}
		Expect(k8sClient.Create(ctx, run)).Should(Succeed())
		Eventually(func(g Gomega) {
			By("Getting ApprovalTask")
			task := &customTasksApi.ApprovalTask{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: makeApprovalTaskName(run), Namespace: ns}, task)
			g.Expect(err).ShouldNot(HaveOccurred())

			By("Setting ApprovalTask action to " + action)
			task.Spec.Action = action
			task.Spec.Approve = &customTasksApi.Approve{
				ApprovedBy: "admin",
				Comment:    comment,
			}
			g.Expect(k8sClient.Update(ctx, task)).Should(Succeed())
		}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
		By("Checking CustomRun completion")
		Eventually(func(g Gomega) {
			By("Getting CustomRun")
			createdRun := &tektonPipelineApi.CustomRun{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: run.Name, Namespace: ns}, createdRun)

			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdRun.IsSuccessful()).Should(Equal(shouldSucceed))
			g.Expect(createdRun.Status.CompletionTime).ShouldNot(BeNil())
			g.Expect(createdRun.Status.Results).Should(HaveLen(3))
			g.Expect(createdRun.Status.Results[0].Value).Should(Equal(approvedValue))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	}
	It("Should process CustomRun with approve ApprovalTask", func() {
		testApprovalTaskAction("test-custom-run", customTasksApi.TaskApproved, "ok", "true", true)
	})
	It("Should process CustomRun with reject ApprovalTask", func() {
		testApprovalTaskAction("test-custom-run-with-reject", customTasksApi.TaskRejected, "not ok", "false", false)
	})
	It("Should cancel ApprovalTask after canceling CustomRun", func() {
		By("Creating CustomRun object")
		run := &tektonPipelineApi.CustomRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-custom-run-canceled",
				Namespace: ns,
			},
			Spec: tektonPipelineApi.CustomRunSpec{
				CustomRef: &tektonPipelineApi.TaskRef{
					APIVersion: customTasksApi.GroupVersion.String(),
					Kind:       customTasksApi.ApprovalTaskKind,
				},
			},
		}
		Expect(k8sClient.Create(ctx, run)).Should(Succeed())
		By("Waiting for ApprovalTask creation")
		Eventually(func(g Gomega) {
			task := &customTasksApi.ApprovalTask{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: makeApprovalTaskName(run), Namespace: ns}, task)
			g.Expect(err).ShouldNot(HaveOccurred())
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
		By("Canceling CustomRun")
		Eventually(func(g Gomega) {
			createdRun := &tektonPipelineApi.CustomRun{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: run.Name, Namespace: ns}, createdRun)
			g.Expect(err).ShouldNot(HaveOccurred())

			createdRun.Spec.Status = tektonPipelineApi.CustomRunSpecStatusCancelled
			g.Expect(k8sClient.Update(ctx, createdRun)).Should(Succeed())
		}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
		By("Checking ApprovalTask cancellation")
		Eventually(func(g Gomega) {
			By("Getting ApprovalTask")
			task := &customTasksApi.ApprovalTask{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: makeApprovalTaskName(run), Namespace: ns}, task)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(task.Spec.Action).Should(Equal(customTasksApi.TaskCanceled))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
		By("Checking CustomRun completion")
		Eventually(func(g Gomega) {
			By("Getting CustomRun")
			createdRun := &tektonPipelineApi.CustomRun{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: run.Name, Namespace: ns}, createdRun)

			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdRun.IsSuccessful()).Should(BeFalse())
			g.Expect(createdRun.Status.CompletionTime).ShouldNot(BeNil())
			g.Expect(createdRun.Status.Results).Should(HaveLen(1))
			g.Expect(createdRun.Status.Results[0].Value).Should(Equal("false"))
		}).WithTimeout(time.Second * 20).WithPolling(time.Second).Should(Succeed())
	})
	It("Should skip CustomRun without ApprovalTask ref", func() {
		By("Creating CustomRun object")
		run := &tektonPipelineApi.CustomRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-custom-run-without-approval-task-ref",
				Namespace: ns,
			},
			Spec: tektonPipelineApi.CustomRunSpec{
				CustomRef: &tektonPipelineApi.TaskRef{
					APIVersion: "v1",
					Kind:       "Test",
				},
			},
		}
		Expect(k8sClient.Create(ctx, run)).Should(Succeed())
		By("Checking ApprovalTask not created")
		Consistently(func(g Gomega) {
			task := &customTasksApi.ApprovalTask{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: makeApprovalTaskName(run), Namespace: ns}, task)
			g.Expect(err).Should(HaveOccurred())
			g.Expect(k8sErrors.IsNotFound(err)).Should(BeTrue())
		}, time.Second*15).Should(Succeed())
	})
})
