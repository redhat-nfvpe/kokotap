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
      --ifname="mirror"          Mirror interface name
      --vxlan-id=VXLAN-ID        VxLAN ID to encap tap traffic
      --mirrortype="both"         mirroring type {ingress|egress|both}
      --dest-node=DEST-NODE      kubernetes node for tap interface
      --namespace="default"      namespace for pod/container (optional)
      --kubeconfig=KUBECONFIG    kubeconfig file path (optional)
```
## Example1 - Create a mirror interface for Pod 'centos'

```
[centos@kube-master ~]$ ./kokotap --pod=centos --mirrortype=both \
    --dest-node=kube-master --vxlan-id=100 | kubectl create -f -
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

### Delete mirror interface

```
[centos@kube-master ~]$ ./kokotap --pod=centos --mirrortype=both \
    --dest-node=kube-master --vxlan-id=100 | kubectl delete -f -
pod "kokotap-centos-sender" deleted
pod "kokotap-centos-receiver-kube-master" deleted
[centos@kube-master ~]$ ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
(snip)
```

You can also delete mirror interface by removing two pods (begins with 'kokotap-', find by 'kubectl get pod')

# Todo
- Add more usable feature (logging?)
- Document

# Authors
- Tomofumi Hayashi (s1061123)
