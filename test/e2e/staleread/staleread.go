package staleread

import (
	"context"
	"time"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2ekubelet "k8s.io/kubernetes/test/e2e/framework/kubelet"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"

	"github.com/onsi/ginkgo"

	// ensure libs have a chance to initialize
	_ "github.com/stretchr/testify/assert"
)

var masterNodes sets.String

var _ = SIGDescribe("Stale read test", func() {
	var cs clientset.Interface
	var nodeList *v1.NodeList
	var ns string
	f := framework.NewDefaultFramework("stale-read")

	ginkgo.AfterEach(func() {
		framework.Logf("after test")
	})

	ginkgo.BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace.Name
		nodeList = &v1.NodeList{}
		var err error

		framework.AllNodesReady(cs, time.Minute)

		masterNodes, _, err = e2enode.GetMasterAndWorkerNodes(cs)
		if err != nil {
			framework.Logf("Unexpected error occurred: %v", err)
		}
		nodeList, err = e2enode.GetReadySchedulableNodes(cs)
		if err != nil {
			framework.Logf("Unexpected error occurred: %v", err)
		}

		framework.ExpectNoErrorWithOffset(0, err)

		err = framework.CheckTestingNSDeletedExcept(cs, ns)
		framework.ExpectNoError(err)

		for _, node := range nodeList.Items {
			framework.Logf("\nLogging pods the kubelet thinks is on node %v before test", node.Name)
			printAllKubeletPods(cs, node.Name)
		}
	})

	framework.ConformanceIt("delete kind-worker", func() {
		WaitForStableCluster(cs, masterNodes)

		err := cs.CoreV1().Nodes().Delete(context.TODO(), "kind-worker", metav1.DeleteOptions{})
		framework.ExpectNoError(err)
		framework.Logf("Delete node kind-worker.")
		time.Sleep(5 * time.Second)

		deployment := &apps.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "demo-deployment",
			},
			Spec: apps.DeploymentSpec{
				Replicas: int32Ptr(6),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "demo",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "demo",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "web",
								Image: "nginx:1.12",
								Ports: []v1.ContainerPort{
									{
										Name:          "http",
										Protocol:      v1.ProtocolTCP,
										ContainerPort: 80,
									},
								},
							},
						},
					},
				},
			},
		}
		deploymentsClient := cs.AppsV1().Deployments(v1.NamespaceDefault)
		podsClient := cs.CoreV1().Pods(v1.NamespaceDefault)
		result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		framework.Logf("Created deployment %q.", result.GetObjectMeta().GetName())
		time.Sleep(5 * time.Second)

		list, err := podsClient.List(context.TODO(), metav1.ListOptions{})
		framework.ExpectNoError(err)
		for _, d := range list.Items {
			framework.Logf("pod: %v", d)
		}

		derr := deploymentsClient.Delete(context.TODO(), "demo-deployment", metav1.DeleteOptions{})
		framework.ExpectNoError(derr)
		framework.Logf("Deleted deployment demo-deployment.")
	})

})

func int32Ptr(i int32) *int32 { return &i }

// printAllKubeletPods outputs status of all kubelet pods into log.
func printAllKubeletPods(c clientset.Interface, nodeName string) {
	podList, err := e2ekubelet.GetKubeletPods(c, nodeName)
	if err != nil {
		framework.Logf("Unable to retrieve kubelet pods for node %v: %v", nodeName, err)
		return
	}
	for _, p := range podList.Items {
		framework.Logf("%v from %v started at %v (%d container statuses recorded)", p.Name, p.Namespace, p.Status.StartTime, len(p.Status.ContainerStatuses))
		for _, c := range p.Status.ContainerStatuses {
			framework.Logf("\tContainer %v ready: %v, restart count %v",
				c.Name, c.Ready, c.RestartCount)
		}
	}
}
