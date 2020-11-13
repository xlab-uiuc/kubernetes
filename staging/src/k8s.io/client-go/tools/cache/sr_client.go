package cache

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

var (
	mu             sync.Mutex
	serverHostPort string = "localhost:1234"
)

// SRSendDeleteNode sends delete node to server
func SRSendDeleteNode(node string) {
	mu.Lock()
	defer mu.Unlock()
	SRSend(serverHostPort, "[SR]\tDN\t"+node)
}

// SRSendDeletePod sends detete pod to server
func SRSendDeletePod(pod *v1.Pod) {
	mu.Lock()
	defer mu.Unlock()
	if pod.Namespace == "kube-system" || pod.Namespace == "local-path-storage" {
		return
	}
	klog.Warningf("[SR] prepare to delete pod %s %s", pod.Name, pod.Namespace)
	SRSend(serverHostPort, "[SR]\tDP\t"+pod.Name+"\t"+pod.Namespace)
}

// SRSendDeleteRS sends detete replicaset to server
func SRSendDeleteRS(rs *apps.ReplicaSet) {
	mu.Lock()
	defer mu.Unlock()
	if rs.Namespace == "kube-system" || rs.Namespace == "local-path-storage" {
		return
	}
	klog.Warningf("[SR] prepare to delete replicaset %s %s", rs.Name, rs.Namespace)
	SRSend(serverHostPort, "[SR]\tDRS\t"+rs.Name+"\t"+rs.Namespace)
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
	if reply == "ACK" {
		klog.Warningf("[SR] make stale successfully")
		return true
	}
	return false
}
