/*
*/
package main

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	//"net"
	"os"
	"path/filepath"
	"strings"
)

// VERSION indicates kokotap's version.
var VERSION = "master@git"

type KokotapArgs struct {
	Pod string
	Namespace string // optional
	Container string //optional
	PodIFName string //optional
	DestNode string
	DestIFName string
	MirrorType string
	VxlanID int
	KubeConfig string // optional
}

type KokotapPodArgs struct {
	ContainerRuntime string
	VxlanID int
	IFName string
	Sender struct {
		ContainerID string
		MirrorType string
		MirrorIF string
		VxlanEgressIP string
		VxlanIP string
	}
	Receiver struct {
		VxlanEgressIP string
		VxlanIP string
	}
}

func (podargs *KokotapPodArgs) GenerateDockerYaml() (string) {
	kokoTapPodDockerTemplate := `
---
apiVersion: v1
kind: Pod
metadata:
  name: koko-sidecar-sender
spec:
  hostNetwork: true
  nodeName: kube-node-2
  containers:
    - name: koko-sidecar-sender
      image: docker.io/s1061123/kokotap:latest
      command: ["/bin/kokotap_pod"]
      args: ["--procprefix=/host", "mode", "sender", "--containerid=%s",
             "--mirrortype=%s", "--mirrorif=%s", "--ifname=%s",
             "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d"]
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
---
apiVersion: v1
kind: Pod
metadata:
  name: koko-sidecar-receiver
spec:
  hostNetwork: true
  nodeName: kube-master
  containers:
    - name: koko-sidecar-receiver
      image: docker.io/s1061123/kokotap:latest
      command: ["/bin/kokotap_pod"]
      args: ["--procprefix=/host", "mode", "receiver",
             "--ifname=%s", "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d"]
      securityContext:
        privileged: true
`
	return fmt.Sprintf(kokoTapPodDockerTemplate, podargs.Sender.ContainerID,
		podargs.Sender.MirrorType, podargs.Sender.MirrorIF, podargs.IFName,
		podargs.Sender.VxlanEgressIP, podargs.Sender.VxlanIP, podargs.VxlanID,
		podargs.IFName, podargs.Receiver.VxlanEgressIP, podargs.Receiver.VxlanIP, podargs.VxlanID)
}

func (podargs *KokotapPodArgs) GenerateCrioYaml() (string) {
	kokoTapPodCrioTemplate := `
---
apiVersion: v1
kind: Pod
metadata:
  name: koko-sidecar-sender
spec:
  hostNetwork: true
  nodeName: kube-node-2
  containers:
    - name: koko-sidecar-sender
      image: docker.io/s1061123/kokotap:latest
      command: ["/bin/kokotap"]
      args: ["--procprefix=/host", "mode", "sender", "--containerid=%s",
             "--mirrortype=%s", "--mirrorif=%s", "--ifname=%s",
             "--vxlan-egressip=%s", "--vxlan-ip=%s", "--vxlan-id=%d"]
      securityContext:
        privileged: true
      volumeMounts:
      - name: var-crio
        mountPath: /var/run/crio/crio.sock
      - name: proc
        mountPath: /host/proc
  volumes:
    - name: var-docker
      hostPath:
        path: /var/run/crio/crio.sock
    - name: proc
      hostPath:
        path: /proc
---
apiVersion: v1
kind: Pod
metadata:
  name: koko-sidecar-receiver
spec:
  hostNetwork: true
  nodeName: kube-master
  containers:
    - name: koko-sidecar-receiver
      image: docker.io/s1061123/kokotap:latest
      command: ["/bin/kokotap"]
      args: ["--procprefix=/host", "mode", "receiver",
             "--ifname=%s", "--vxlan-egressif=%s", "--vxlan-ip=%s", "--vxlan-id=%d"]
      securityContext:
        privileged: true
`
	return fmt.Sprintf(kokoTapPodCrioTemplate, podargs.Sender.ContainerID,
		podargs.Sender.MirrorType, podargs.Sender.MirrorIF, podargs.IFName,
		podargs.Sender.VxlanEgressIP, podargs.Sender.VxlanIP, podargs.VxlanID,
		podargs.IFName, podargs.Receiver.VxlanEgressIP, podargs.Receiver.VxlanIP, podargs.VxlanID)
}

func (podargs *KokotapPodArgs) ParseKokoTapArgs(args *KokotapArgs) error {
	if args == nil {
		return fmt.Errorf("Invalid args")
	}


	kubeClient, err := GetK8sClient(args.KubeConfig, nil)
	if err != nil {
		return fmt.Errorf("err:%v", err)
	}
	pod, err := kubeClient.GetPod(args.Namespace, args.Pod)
	if err != nil {
		return fmt.Errorf("err:%v", err)
	}
	podargs.Sender.VxlanEgressIP = pod.Status.HostIP
	podargs.Receiver.VxlanIP = pod.Status.HostIP

	isContainerFound := false
	for _, val := range pod.Status.ContainerStatuses {
		if val.Name == args.Container {
			podargs.Sender.ContainerID = val.ContainerID
			isContainerFound = true
			break
		}
	}
	if isContainerFound != true {
		return fmt.Errorf("container: %s is not found", args.Container)
	}

	podargs.ContainerRuntime = podargs.Sender.
		ContainerID[0:strings.Index(podargs.Sender.ContainerID, ":")]
	podargs.IFName = "mirror"
	podargs.Sender.MirrorType = args.MirrorType
	podargs.Sender.MirrorIF = args.PodIFName
	podargs.VxlanID = args.VxlanID

	destNode, err := kubeClient.GetNode(args.DestNode)
        if err != nil {
		return fmt.Errorf("err:%v", err)
        }
	_, destIP := GetHostIP(&destNode.Status.Addresses)
	podargs.Receiver.VxlanEgressIP = destIP
	podargs.Sender.VxlanIP = destIP

	return nil
}

func (args *KokotapArgs) fillOptionalArgs() {
	if args.Container == "" {
		args.Container = args.Pod
	}
}

func main() {
	var args KokotapArgs
	a := kingpin.New(filepath.Base(os.Args[0]), "kokotap_pod")
	a.Version(VERSION)

	a.HelpFlag.Short('h')
	k := a.Command("create", "create tap interface for kubernetes pod")
	k.Flag("pod", "tap target pod name").Required().StringVar(&args.Pod)
	k.Flag("pod-ifname", "tap target interface name of pod (optional)").
		Default("eth0").StringVar(&args.PodIFName)
	k.Flag("pod-container", "tap target container name (optional)").
		StringVar(&args.Container)
	k.Flag("vxlan-id", "VxLAN ID to encap tap traffic").
		Required().IntVar(&args.VxlanID)
	k.Flag("mirrortype", "mirroring type {ingress|egress|all}").
		Default("all").StringVar(&args.MirrorType)
	k.Flag("dest-node", "kubernetes node for tap interface").Required().StringVar(&args.DestNode)
	k.Flag("dest-ifname", "tap interface name").Required().StringVar(&args.DestIFName)
	k.Flag("namespace", "namespace for pod/container (optional)").
		Default("default").StringVar(&args.Namespace)
	k.Flag("kubeconfig", "kubeconfig file path (optional)").
		Envar("KUBECONFIG").StringVar(&args.KubeConfig)

	switch kingpin.MustParse(a.Parse(os.Args[1:])) {
	case k.FullCommand():
		args.fillOptionalArgs()
		//fmt.Printf("args: %+v\n", args)
	}

	podArgs := KokotapPodArgs{}
	err := podArgs.ParseKokoTapArgs(&args)
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
	}
	//fmt.Printf("podArgs: %+v\n", podArgs)
	
	switch podArgs.ContainerRuntime {
	case "docker":
		fmt.Printf("%s", podArgs.GenerateDockerYaml())
	case "cri-o":
		fmt.Printf("%s", podArgs.GenerateCrioYaml())
	}
	return
}
