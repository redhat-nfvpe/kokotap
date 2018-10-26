/*
*/
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"path/filepath"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"strings"
	koko "github.com/redhat-nfvpe/koko/api"
)

// VERSION indicates kokotap's version.
var VERSION = "master@git"

type SenderArgs struct {
	ContainerID string
	MirrorType string
	MirrorIfName string
	IfName string
	VxlanEgressIf string
	VxlanEgressIP string
	VxlanID int
	VxlanIP net.IP
}

type ReceiverArgs struct {
	IfName string
	VxlanEgressIf string
	VxlanEgressIP string
	VxlanID int
	VxlanIP net.IP
}

func getInterfaceByAddr(addr string) (*net.Interface, error) {
        ifs, err := net.Interfaces()
        if err != nil {
                return nil, err
        }

        for _, i := range ifs {
                addrs, err := i.Addrs()
                if err != nil {
                        return nil, err
                }
		// TODO: need to refine for optimization
                for _, a := range addrs {
                        ip, _, _ := net.ParseCIDR(a.String())
                        if ip.String() == addr {
                                return &i, err
                        }
                }
        }
        return nil, err
}

func parseSenderArgs(procPrefix string, args *SenderArgs) (*koko.VEth, *koko.VxLan, error) {
	var err error
	veth := koko.VEth{}

	if args.VxlanEgressIP != "" {
		egressif, err := getInterfaceByAddr(args.VxlanEgressIP)
		if err != nil {
			return nil, nil, err
		}
		args.VxlanEgressIf = egressif.Name	
	}

	containerType := args.ContainerID[0:strings.Index(args.ContainerID, ":")]
	containerID := args.ContainerID[strings.Index(args.ContainerID, "://")+3:]
	switch containerType {
	case "cri-o":
		veth.NsName, err = koko.GetCrioContainerNS(procPrefix, containerID, "")
	case "docker":
		veth.NsName, err = koko.GetDockerContainerNS(procPrefix, containerID)
	}
	if err != nil {
		return nil, nil, err
	}

	exists, _ := koko.IsExistLinkInNS(veth.NsName, args.IfName)
	if exists == true {
		return nil, nil, fmt.Errorf("XXX")
	}
	veth.LinkName = args.IfName

	switch args.MirrorType {
	case "ingress":
		veth.MirrorIngress = args.MirrorIfName
	case "egress":
		veth.MirrorEgress = args.MirrorIfName
	case "both":
		veth.MirrorIngress = args.MirrorIfName
		veth.MirrorEgress = args.MirrorIfName
	}

	vxlan := koko.VxLan{}
	vxlan.ParentIF = args.VxlanEgressIf
	vxlan.IPAddr = args.VxlanIP
	vxlan.ID = args.VxlanID
	return &veth, &vxlan, nil
}

func parseReceiverArgs(procPrefix string, args *ReceiverArgs) (*koko.VEth, *koko.VxLan, error) {
	exists, _ := koko.IsExistLinkInNS("", args.IfName)
	if exists == true {
		return nil, nil, fmt.Errorf("XXX")
	}

	if args.VxlanEgressIP != "" {
		egressif, err := getInterfaceByAddr(args.VxlanEgressIP)
		if err != nil {
			return nil, nil, err
		}
		args.VxlanEgressIf = egressif.Name	
	}

	veth := koko.VEth{}
	veth.NsName = ""
	veth.LinkName = args.IfName

	vxlan := koko.VxLan{}
	vxlan.ParentIF = args.VxlanEgressIf
	vxlan.IPAddr = args.VxlanIP
	vxlan.ID = args.VxlanID
	return &veth, &vxlan, nil
}

func main() {
	a := kingpin.New(filepath.Base(os.Args[0]), "kokotap")
	a.Version(VERSION)

	var senderArgs SenderArgs
	var receiverArgs ReceiverArgs
	var procPrefix string
	var _ koko.VxLan

	a.HelpFlag.Short('h')
	a.Flag("procprefix", "prefix for /proc filesystem").StringVar(&procPrefix)
	//a.Flag("mode", "Kokotap mode (sender/receiver)").StringVar(&mode)
	k := a.Command("mode", "Kokotap mode (sender/receiver)")
	s := k.Command("sender", "sender mode")
	s.Flag("containerid", "container id").
		Required().StringVar(&senderArgs.ContainerID)
	s.Flag("mirrortype", "mirror type (ingress)").
		Required().StringVar(&senderArgs.MirrorType)
	s.Flag("mirrorif", "mirror target interface").
		Required().StringVar(&senderArgs.MirrorIfName)
	s.Flag("ifname", "interface name for container").
		Required().StringVar(&senderArgs.IfName)
	s.Flag("vxlan-egressif", "Egress interface for vxlan").
		StringVar(&senderArgs.VxlanEgressIf)
	s.Flag("vxlan-egressip", "Egress interface ip address for vxlan").
		StringVar(&senderArgs.VxlanEgressIP)
	s.Flag("vxlan-id", "Vxlan ID").
		Required().IntVar(&senderArgs.VxlanID)
	s.Flag("vxlan-ip", "Vxlan neighbor IP").
		Required().IPVar(&senderArgs.VxlanIP)

	r := k.Command("receiver", "receiver mode")
	r.Flag("ifname", "interface name").
		Required().StringVar(&receiverArgs.IfName)
	r.Flag("vxlan-egressif", "Egress interface for vxlan").
		StringVar(&receiverArgs.VxlanEgressIf)
	r.Flag("vxlan-egressip", "Egress interface ip addressfor vxlan").
		StringVar(&receiverArgs.VxlanEgressIP)
	r.Flag("vxlan-id", "Vxlan ID").
		Required().IntVar(&receiverArgs.VxlanID)
	r.Flag("vxlan-ip", "Vxlan neighbor IP").
		Required().IPVar(&receiverArgs.VxlanIP)

	var veth *koko.VEth
	var vxlan *koko.VxLan
	var err error

	switch kingpin.MustParse(a.Parse(os.Args[1:])) {
	case s.FullCommand():
		fmt.Printf("sender\n")
		veth, vxlan, err = parseSenderArgs(procPrefix, &senderArgs)

	case r.FullCommand():
		fmt.Printf("receiver\n")
		veth, vxlan, err = parseReceiverArgs(procPrefix, &receiverArgs)
	}

	sig := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Printf("\nCatch signal!\n")
		done <- true
	}()

	err = koko.MakeVxLan(*veth, *vxlan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		//bailout?
	}
	fmt.Println("Waiting for signal at main ...")
	<-done
	err = veth.RemoveVethLink()
	if err != nil {
		fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		//bailout?
	}
	fmt.Println("Exit from main")
}
