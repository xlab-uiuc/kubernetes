package sr

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clientset "k8s.io/client-go/kubernetes"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework"
)

var (
	timeout  = 10 * time.Minute
	waitTime = 2 * time.Second
)

// SIGDescribe annotates the test with the SIG label.
func SIGDescribe(text string, body func()) bool {
	return ginkgo.Describe("[stale-read] "+text, body)
}

// WaitForStableCluster waits until all existing pods are scheduled and returns their amount.
func WaitForStableCluster(c clientset.Interface, masterNodes sets.String) int {
	startTime := time.Now()
	// Wait for all pods to be scheduled.
	allScheduledPods, allNotScheduledPods := getScheduledAndUnscheduledPods(c, masterNodes, metav1.NamespaceAll)
	for len(allNotScheduledPods) != 0 {
		time.Sleep(waitTime)
		if startTime.Add(timeout).Before(time.Now()) {
			framework.Logf("Timed out waiting for the following pods to schedule")
			for _, p := range allNotScheduledPods {
				framework.Logf("%v/%v", p.Namespace, p.Name)
			}
			framework.Failf("Timed out after %v waiting for stable cluster.", timeout)
			break
		}
		allScheduledPods, allNotScheduledPods = getScheduledAndUnscheduledPods(c, masterNodes, metav1.NamespaceAll)
	}
	return len(allScheduledPods)
}

// WaitForPodsToBeDeleted waits until pods that are terminating to get deleted.
func WaitForPodsToBeDeleted(c clientset.Interface) {
	startTime := time.Now()
	deleting := getDeletingPods(c, metav1.NamespaceAll)
	for len(deleting) != 0 {
		if startTime.Add(timeout).Before(time.Now()) {
			framework.Logf("Pods still not deleted")
			for _, p := range deleting {
				framework.Logf("%v/%v", p.Namespace, p.Name)
			}
			framework.Failf("Timed out after %v waiting for pods to be deleted", timeout)
			break
		}
		time.Sleep(waitTime)
		deleting = getDeletingPods(c, metav1.NamespaceAll)
	}
}

// getScheduledAndUnscheduledPods lists scheduled and not scheduled pods in the given namespace, with succeeded and failed pods filtered out.
func getScheduledAndUnscheduledPods(c clientset.Interface, masterNodes sets.String, ns string) (scheduledPods, notScheduledPods []v1.Pod) {
	pods, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("listing all pods in namespace %q while waiting for stable cluster", ns))
	// API server returns also Pods that succeeded. We need to filter them out.
	filteredPods := make([]v1.Pod, 0, len(pods.Items))
	for _, p := range pods.Items {
		if !podTerminated(p) {
			filteredPods = append(filteredPods, p)
		}
	}
	pods.Items = filteredPods
	return GetPodsScheduled(masterNodes, pods)
}

// getDeletingPods returns whether there are any pods marked for deletion.
func getDeletingPods(c clientset.Interface, ns string) []v1.Pod {
	pods, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("listing all pods in namespace %q while waiting for pods to terminate", ns))
	var deleting []v1.Pod
	for _, p := range pods.Items {
		if p.ObjectMeta.DeletionTimestamp != nil && !podTerminated(p) {
			deleting = append(deleting, p)
		}
	}
	return deleting
}

func podTerminated(p v1.Pod) bool {
	return p.Status.Phase == v1.PodSucceeded || p.Status.Phase == v1.PodFailed
}

// GetPodsScheduled returns a number of currently scheduled and not scheduled Pods.
func GetPodsScheduled(masterNodes sets.String, pods *v1.PodList) (scheduledPods, notScheduledPods []v1.Pod) {
	for _, pod := range pods.Items {
		if !masterNodes.Has(pod.Spec.NodeName) {
			if pod.Spec.NodeName != "" {
				_, scheduledCondition := podutil.GetPodCondition(&pod.Status, v1.PodScheduled)
				framework.ExpectEqual(scheduledCondition != nil, true)
				framework.ExpectEqual(scheduledCondition.Status, v1.ConditionTrue)
				scheduledPods = append(scheduledPods, pod)
			} else {
				_, scheduledCondition := podutil.GetPodCondition(&pod.Status, v1.PodScheduled)
				framework.ExpectEqual(scheduledCondition != nil, true)
				framework.ExpectEqual(scheduledCondition.Status, v1.ConditionFalse)
				if scheduledCondition.Reason == "Unschedulable" {

					notScheduledPods = append(notScheduledPods, pod)
				}
			}
		}
	}
	return
}
