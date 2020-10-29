package cache

import (
	"bufio"
	"fmt"
	"net"

	"k8s.io/klog"
)

// SRSend sends text to hostPort
func SRSend(hostPort string, text string) {
	c, err := net.Dial("tcp", hostPort)
	if err != nil {
		klog.Errorf("[SR] failed to connect to SRServer: %v", err)
		return
	}
	klog.Warningf("[SR] send %s to %s", text, hostPort)
	fmt.Fprintf(c, text+"\n")
	message, _ := bufio.NewReader(c).ReadString('\n')
	klog.Warningf("[SR] hear back: %s", message)
}
