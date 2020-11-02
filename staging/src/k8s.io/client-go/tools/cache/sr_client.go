package cache

import (
	"bufio"
	"fmt"
	"net"

	"k8s.io/klog"
)

var (
	serverHostPort string = "localhost:1234"
	// DeleteNodeScheduler controls whether delete the node at scheduler
	DeleteNodeScheduler bool = true
)

// SRSendDeleteNode sends delete node to server
func SRSendDeleteNode(node string) {
	SRSend(serverHostPort, "[SR]\tDN\t"+node)
}

// SRSend sends text to server
func SRSend(hostPort string, text string) {
	c, connectErr := net.Dial("tcp", hostPort)
	if connectErr != nil {
		klog.Errorf("[SR] failed to connect to SRServer: %v", connectErr)
		return
	}
	klog.Warningf("[SR] send %s to %s", text, hostPort)
	fmt.Fprintf(c, text+"\n")
	reply, readErr := bufio.NewReader(c).ReadString('\n')
	if readErr != nil {
		klog.Errorf("[SR] failed to hear back from SRServer: %v", readErr)
		return
	}
	klog.Warningf("[SR] hear back: %s", reply)
}
