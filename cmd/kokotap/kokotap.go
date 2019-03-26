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
 * kokotap main code
 */
import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var version = "master@git"
var commit = "unknown commit"
var date = "unknown date"

type kokotapArgs struct {
	Pod        string
	Namespace  string // optional
	Container  string // optional
	PodIFName  string // optional
	IFName     string // optional (ifname for tapping if)
	DestNode   string
	DestIP     net.IP
	MirrorType string
	VxlanID    int
	VxlanPort  int    // UDP port, optional
	KubeConfig string // optional
	Image      string // optional
}

type kokotapPodArgs struct {
	ContainerRuntime string
	PodName          string
	VxlanID          int
	IFName           string
	Sender           struct {
		Node          string
		ContainerID   string
		MirrorType    string
		MirrorIF      string
		VxlanEgressIP string // Egress IF's IP
		VxlanIP       string // Dest Vxlan IP
	}
	Receiver struct {
		Node          string
		VxlanEgressIP string // Egress IF's IP
		VxlanIP       string // Dest Vxlan IP
	}
	Image string
}

func (podargs *kokotapPodArgs) GeneratePodName() (string, string) {
	nodeName := strings.Replace(podargs.Receiver.Node, ".", "-", -1)
	return fmt.Sprintf("kokotap-%s-sender", podargs.PodName),
		fmt.Sprintf("kokotap-%s-receiver-%s", podargs.PodName, nodeName)
}

func (podargs *kokotapPodArgs) GenerateDockerYaml() string {
	senderPod, receiverPod := podargs.GeneratePodName()
	kokoTapPodDockerSenderTemplate := `
---
apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  hostNetwork: true
  nodeName: %s
  containers:
    - name: %s
      image: %s
      command: ["/bin/kokotap_pod"]
      args: ["--procprefix=/host", "mode", "sender", "--containerid=%s",
             "--mirrortype=%s", "--mirrorif=%s", "--ifname=%s",
             "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d",
             "--vxlan-port=%d"]
      securityContext:
        privileged: true
      volumeMounts:
      - name: var-docker
        mountPath: /var/run/docker.sock
      - name: proc
        mountPath: /host/proc
  volumes:
    - name: var-docker
      hostPath:
        path: /var/run/docker.sock
    - name: proc
      hostPath:
        path: /proc
`
	kokoTapPodDockerReceiverTemplate := `
---
apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  hostNetwork: true
  nodeName: %s
  containers:
    - name: %s
      image: %s
      command: ["/bin/kokotap_pod"]
      args: ["--procprefix=/host", "mode", "receiver",
             "--ifname=%s", "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d",
             "--vxlan-port=%d"]
      securityContext:
        privileged: true
`
	yaml := fmt.Sprintf(kokoTapPodDockerSenderTemplate,
		senderPod, podargs.Sender.Node, senderPod, podargs.Image,
		podargs.Sender.ContainerID,
		podargs.Sender.MirrorType, podargs.Sender.MirrorIF, podargs.IFName,
		podargs.Sender.VxlanEgressIP, podargs.Sender.VxlanIP, podargs.VxlanID)

	if podargs.Receiver.Node != "" {
		yaml = yaml + fmt.Sprintf(kokoTapPodDockerReceiverTemplate,
			receiverPod, podargs.Receiver.Node, receiverPod, podargs.Image,
			podargs.IFName, podargs.Receiver.VxlanEgressIP, podargs.Receiver.VxlanIP, podargs.VxlanID)
	}

	return yaml
}

func (podargs *kokotapPodArgs) GenerateCrioYaml() string {
	senderPod, receiverPod := podargs.GeneratePodName()
	kokoTapPodCrioSenderTemplate := `
---
apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  hostNetwork: true
  nodeName: %s
  containers:
    - name: %s
      image: %s
      command: ["/bin/kokotap_pod"]
      args: ["--procprefix=/host", "mode", "sender", "--containerid=%s",
             "--mirrortype=%s", "--mirrorif=%s", "--ifname=%s",
             "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d",
             "--vxlan-port=%d"]
      securityContext:
        privileged: true
      volumeMounts:
      - name: var-crio
        mountPath: /var/run/crio/crio.sock
      - name: proc
        mountPath: /host/proc
  volumes:
    - name: var-crio
      hostPath:
        path: /var/run/crio/crio.sock
    - name: proc
      hostPath:
        path: /proc
`
	kokoTapPodCrioReceiverTemplate := `
---
apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  hostNetwork: true
  nodeName: %s
  containers:
    - name: %s
      image: %s
      command: ["/bin/kokotap_pod"]
      args: ["--procprefix=/host", "mode", "receiver",
             "--ifname=%s", "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d",
             "--vxlan-port=%d"]
      securityContext:
        privileged: true
`
	yaml := fmt.Sprintf(kokoTapPodCrioSenderTemplate,
		senderPod, podargs.Sender.Node, senderPod, podargs.Image,
		podargs.Sender.ContainerID,
		podargs.Sender.MirrorType, podargs.Sender.MirrorIF, podargs.IFName,
		podargs.Sender.VxlanEgressIP, podargs.Sender.VxlanIP, podargs.VxlanID)

	if podargs.Receiver.Node != "" {
		yaml = yaml + fmt.Sprintf(kokoTapPodCrioReceiverTemplate,
			receiverPod, podargs.Receiver.Node, receiverPod, podargs.Image,
			podargs.IFName, podargs.Receiver.VxlanEgressIP, podargs.Receiver.VxlanIP, podargs.VxlanID)
	}

	return yaml
}

func (podargs *kokotapPodArgs) ParseKokoTapArgs(args *kokotapArgs) error {
	if args == nil {
		return fmt.Errorf("Invalid args")
	}

	if args.KubeConfig == "" {
		return fmt.Errorf("no kubeconfig option")
	}

	_, err := os.Stat(args.KubeConfig)
	if err != nil {
		return fmt.Errorf("kubeconfig %q is not found: %v", args.KubeConfig, err)
	}

	kubeClient, err := getK8sClient(args.KubeConfig, nil)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	pod, err := kubeClient.GetPod(args.Namespace, args.Pod)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	podargs.PodName = args.Pod
	podargs.Sender.VxlanEgressIP = pod.Status.HostIP
	podargs.Receiver.VxlanIP = pod.Status.HostIP
	podargs.Sender.Node = pod.Spec.NodeName
	podargs.Image = args.Image

	isContainerFound := false
	for _, val := range pod.Status.ContainerStatuses {
		if val.Ready == true {
			podargs.Sender.ContainerID = val.ContainerID
			isContainerFound = true
			break
		}
	}
	if isContainerFound != true {
		return fmt.Errorf("no ready container in pod: %q", args.Pod)
	}

	podargs.ContainerRuntime = podargs.Sender.
		ContainerID[0:strings.Index(podargs.Sender.ContainerID, ":")]
	podargs.IFName = args.IFName
	podargs.Sender.MirrorType = args.MirrorType
	podargs.Sender.MirrorIF = args.PodIFName
	podargs.VxlanID = args.VxlanID

	if args.DestNode != "" && args.DestIP == nil {
		destNode, err := kubeClient.GetNode(args.DestNode)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		destNodeName, destIP := getHostIP(&destNode.Status.Addresses)
		podargs.Receiver.VxlanEgressIP = destIP
		podargs.Sender.VxlanIP = destIP
		podargs.Receiver.Node = destNodeName
	} else if args.DestNode == "" && args.DestIP != nil {
		podargs.Receiver.VxlanEgressIP = string(args.DestIP)
		podargs.Sender.VxlanIP = args.DestIP.String()
	} else {
		return fmt.Errorf("please set dest-node or dest-ip")
	}

	return nil
}

func main() {
	var args kokotapArgs
	/*
		a := kingpin.New(filepath.Base(os.Args[0]), "kokotap_pod")
		a.Version(VERSION)
		a.HelpFlag.Short('h')

		k := a.Command("create", "create tap interface for kubernetes pod")
	*/
	k := kingpin.New(filepath.Base(os.Args[0]), "kokotap")
	k.Version(fmt.Sprintf("%s/%s/%s", version, commit, date))
	k.HelpFlag.Short('h')
	k.VersionFlag.Short('v')

	k.Flag("pod", "tap target pod name").Required().StringVar(&args.Pod)
	k.Flag("pod-ifname", "tap target interface name of pod (optional)").
		Default("eth0").StringVar(&args.PodIFName)
	k.Flag("vxlan-id", "VxLAN ID to encap tap traffic").
		Required().IntVar(&args.VxlanID)
	k.Flag("vxlan-port", "VxLAN UDP port").Default("4789").IntVar(&args.VxlanPort)
	k.Flag("ifname", "Mirror interface name").Default("mirror").StringVar(&args.IFName)
	k.Flag("mirrortype", "mirroring type {ingress|egress|both}").
		Default("both").EnumVar(&args.MirrorType, "ingress", "egress", "both")
	k.Flag("dest-node", "kubernetes node for tap interface").StringVar(&args.DestNode)
	k.Flag("dest-ip", "IP address for destination tap interface").IPVar(&args.DestIP)
	k.Flag("namespace", "namespace for pod/container (optional)").
		Default("default").StringVar(&args.Namespace)
	k.Flag("kubeconfig", "kubeconfig file path (optional)").
		Envar("KUBECONFIG").StringVar(&args.KubeConfig)
	k.Flag("image", "kokotap container image").Default("quay.io/s1061123/kokotap:latest").StringVar(&args.Image)

	kingpin.MustParse(k.Parse(os.Args[1:]))

	podArgs := kokotapPodArgs{}
	err := podArgs.ParseKokoTapArgs(&args)

	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
	}

	switch podArgs.ContainerRuntime {
	case "docker":
		fmt.Printf("%s", podArgs.GenerateDockerYaml())
	case "cri-o":
		fmt.Printf("%s", podArgs.GenerateCrioYaml())
	}
	return
}
