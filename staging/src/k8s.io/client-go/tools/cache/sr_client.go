package cache

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

var (
	mu                  sync.Mutex
	serverHostPort      string = "localhost:1234"
	deleteNodeScheduler bool   = false
	deletePodController bool   = false
)

// SRSendDeleteNode sends delete node to server
func SRSendDeleteNode(node string) {
	mu.Lock()
	defer mu.Unlock()
	if deleteNodeScheduler {
		if SRSend(serverHostPort, "[SR]\tDN\t"+node) {
			deleteNodeScheduler = false
		}
	}
}

// SRSendDeletePod sends detete pod to server
func SRSendDeletePod(pod *v1.Pod) {
	klog.Warning("[SR] delete pod")
	mu.Lock()
	defer mu.Unlock()
	if deletePodController {
		if SRSend(serverHostPort, "[SR]\tDP\t"+pod.Name+"\t"+pod.Namespace) {
			deletePodController = false
		}
	}
}

// SRSend sends text to server
func SRSend(hostPort string, text string) bool {
	c, connectErr := net.Dial("tcp", hostPort)
	if connectErr != nil {
		klog.Errorf("[SR] failed to connect to SRServer: %v", connectErr)
		return false
	}
	klog.Warningf("[SR] send %s to %s", text, hostPort)
	fmt.Fprintf(c, text+"\n")
	reply, readErr := bufio.NewReader(c).ReadString('\n')
	if readErr != nil {
		klog.Errorf("[SR] failed to hear back from SRServer: %v", readErr)
		return false
	}
	klog.Warningf("[SR] hear back: %s", reply)
	if reply == "YES" {
		return true
	}
	return false
}
