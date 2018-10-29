# kokotap: Tapping Pod Traffic to VxLAN interface

# What is 'kokotap'?

`kokotap` provides network tapping for Kubernetes Pod. `kokotap` creates VxLAN interface to target Pod/Container then do packet mirroring to the VxLAN interface by [tc-mirred](http://man7.org/linux/man-pages/man8/tc-mirred.8.html). `kokotap` also creates VxLAN interface to target node (e.g. 'kube-master') to capture the traffic.

# Support

`kokotap` supports following runtime:

- Docker runtime
- cri-o 

# Syntax

Currently `kokotap` creates pod yaml file, so you can put it in `kubectl` to create pods.

Note: Its syntax will be refined soon so please keep in watch...

```
[centos@kube-master ~]$ ./kokotap -h
usage: kokotap --pod=POD --vxlan-id=VXLAN-ID --dest-node=DEST-NODE --dest-ifname=DEST-IFNAME [<flags>]

kokotap

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --version                  Show application version.
      --pod=POD                  tap target pod name
      --pod-ifname="eth0"        tap target interface name of pod (optional)
      --pod-container=POD-CONTAINER  
                                 tap target container name (optional)
      --vxlan-id=VXLAN-ID        VxLAN ID to encap tap traffic
      --mirrortype="all"         mirroring type {ingress|egress|all}
      --dest-node=DEST-NODE      kubernetes node for tap interface
      --dest-ifname=DEST-IFNAME  tap interface name
      --namespace="default"      namespace for pod/container (optional)
      --kubeconfig=KUBECONFIG    kubeconfig file path (optional)
```

## Example1 - Create a mirror interface for Container 'centos-container' in Pod 'centos'

```
[centos@kube-master ~]$ ./kokotap --pod=centos --pod-container=centos-container --mirrortype=both --dest-node=kube-master --dest-ifname=mirror --vxlan-id=100 | kubectl create -f -
pod/kokotap-centos-sender created
pod/kokotap-centos-receiver-kube-master created
[centos@kube-master ~]$ ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
(snip)
17: mirror: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UNKNOWN qlen 1000
    link/ether 7e:3a:cb:bf:95:28 brd ff:ff:ff:ff:ff:ff
    inet6 fe80::7c3a:cbff:febf:9528/64 scope link 
       valid_lft forever preferred_lft forever
```

## Example2 - Create a mirror interface for Pod 'centos'

If your container name is same as pod name, you can skip to input '--pod-container' option.

```
[centos@kube-master ~]$ ./kokotap --pod=centos --mirrortype=both --dest-node=kube-master --dest-ifname=mirror --vxlan-id=100 | kubectl create -f -
pod/kokotap-centos-sender created
pod/kokotap-centos-receiver-kube-master created
[centos@kube-master ~]$ ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
(snip)
17: mirror: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UNKNOWN qlen 1000
    link/ether 7e:3a:cb:bf:95:28 brd ff:ff:ff:ff:ff:ff
    inet6 fe80::7c3a:cbff:febf:9528/64 scope link 
       valid_lft forever preferred_lft forever
```
