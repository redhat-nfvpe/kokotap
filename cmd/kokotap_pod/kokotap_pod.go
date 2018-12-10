// Copyright 2018 Red Hat
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

/*
 * kokotap_pod: kokotap code for pod/container
 */

import (
	"fmt"
	koko "github.com/redhat-nfvpe/koko/api"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var version = "master@git"
var commit = "unknown commit"
var date = "unknown date"

type senderArgs struct {
	ContainerID   string
	MirrorType    string
	MirrorIfName  string
	IfName        string
	VxlanEgressIf string
	VxlanEgressIP string
	VxlanID       int
	VxlanIP       net.IP
}

type receiverArgs struct {
	IfName        string
	VxlanEgressIf string
	VxlanEgressIP string
	VxlanID       int
	VxlanIP       net.IP
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

func parseSenderArgs(procPrefix string, args *senderArgs) (*koko.VEth, *koko.VxLan, error) {
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

func parseReceiverArgs(procPrefix string, args *receiverArgs) (*koko.VEth, *koko.VxLan, error) {
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
	a.Version(fmt.Sprintf("%s/%s/%s", version, commit, date))

	var senderArgs senderArgs
	var receiverArgs receiverArgs
	var procPrefix string
	var _ koko.VxLan

	a.HelpFlag.Short('h')
	a.VersionFlag.Short('v')
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

	var egressMTU int
	var egressTxQLen int
	if veth.MirrorEgress != "" {
		egressMTU, err = koko.GetMTU(veth.MirrorEgress)
		if err != nil {
			fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		}
		egressTxQLen, err = veth.GetEgressTxQLen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		}
	}
	err = koko.MakeVxLan(*veth, *vxlan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		//bailout?
	}

	fmt.Println("Waiting for signal at main ...")
	<-done

	// Cleanup
	if veth.MirrorEgress != "" {
		err = koko.SetMTU(veth.MirrorEgress, egressMTU)
		if err != nil {
			fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		}
		err = veth.SetEgressTxQLen(egressTxQLen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		}
	}
	err = veth.RemoveVethLink()
	if err != nil {
		fmt.Fprintf(os.Stderr, "XXX:%v\n", err)
		//bailout?
	}
	fmt.Println("Exit from main")
}
