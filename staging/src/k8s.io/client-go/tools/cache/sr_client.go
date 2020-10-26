package cache

import (
	"fmt"
	"net"

	"k8s.io/klog"
)

// SRSend sends text to hostPort
func SRSend(hostPort string, text string) {
	c, err := net.Dial("tcp", hostPort)
	if err != nil {
		fmt.Println(err)
		return
	}
	klog.Warningf("[SR] send %s to %s", text, hostPort)
	fmt.Fprintf(c, text+"\n")
	// message, _ := bufio.NewReader(c).ReadString('\n')
	// fmt.Print("[SR] hear back: " + message)
}
