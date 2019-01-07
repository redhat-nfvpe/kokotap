# kokotap: Tapping Pod Traffic to VxLAN interface
[![Travis CI](https://travis-ci.org/redhat-nfvpe/kokotap.svg?branch=master)](https://travis-ci.org/redhat-nfvpe/kokotap/builds)

# What is 'kokotap'?

`kokotap` provides network tapping for Kubernetes Pod. `kokotap` creates VxLAN interface to target Pod/Container then do packet mirroring to the VxLAN interface by [tc-mirred](http://man7.org/linux/man-pages/man8/tc-mirred.8.html). `kokotap` can also create VxLAN interface to Kubernetes target node (e.g. 'kube-master') to capture the traffic or you can specify specific IP addresses for non Kubernetes node for capture.

# Supported Container Runtime

`kokotap` supports following runtime:

- Docker runtime
- cri-o 

# Get Releases
See [releases page](https://github.com/redhat-nfvpe/kokotap/releases).

# Syntax

Currently `kokotap` creates pod yaml file, so you can put it in `kubectl` to create pods.

```
[centos@kube-master ~]$ ./kokotap -h
usage: kokotap --pod=POD --vxlan-id=VXLAN-ID [<flags>]

kokotap

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long and --help-man).
  -v, --version                Show application version.
      --pod=POD                tap target pod name
      --pod-ifname="eth0"      tap target interface name of pod (optional)
      --vxlan-id=VXLAN-ID      VxLAN ID to encap tap traffic
      --ifname="mirror"        Mirror interface name
      --mirrortype=both        mirroring type {ingress|egress|both}
      --dest-node=DEST-NODE    kubernetes node for tap interface
      --dest-ip=DEST-IP        IP address for destination tap interface
      --namespace="default"    namespace for pod/container (optional)
      --kubeconfig=KUBECONFIG  kubeconfig file path (optional)
      --image="quay.io/s1061123/kokotap:latest"
                               kokotap container image
```

## Example1 - Create a mirror interface for Pod 'centos' and receive interface "mirror" at kube-master.

This command creates two interfaces as following:

- VxLAN interface (name: mirror) at Pod to capture eth0 traffic
- VxLAN interface (name: mirror) at the kube-master (container host) to capture above Pod traffic

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

## Example2 - Create a mirror interface for Pod 'centos' (to non-kubernetes node)

This command create an interface as following:

- VxLAN interface (name: mirror) at Pod to capture eth0 traffic

You need to create VxLAN interface manually to receive mirror traffic in this case.

```
[centos@kube-master ~]$ ./kokotap --pod=centos --mirrortype=both \
    --dest-ip=10.1.1.1 --vxlan-id=100 | kubectl create -f -
pod/kokotap-centos-sender created
pod/kokotap-centos-receiver-kube-master created
```

```
[centos@10.1.1.1 ~]$ sudo ip link add mirror type vxlan id 192.168.1.1 dev eth0 dstport 4789
[centos@10.1.1.1 ~]$ sudo ip link set up mirror
```

### Delete mirror interface

Same as Example1, but you need to delete receiver side by hand.

```
[centos@kube-master ~]$ ./kokotap --pod=centos --mirrortype=both \
    --dest-ip=10.1.1.1 --vxlan-id=100 | kubectl delete -f -
pod "kokotap-centos-sender" deleted
pod "kokotap-centos-receiver-kube-master" deleted
(snip)
```

```
[centos@10.1.1.1 ~]$ sudo ip link set down mirror
[centos@10.1.1.1 ~]$ sudo ip link delete mirror
```

You can also delete mirror interface by removing two pods (begins with 'kokotap-', find by 'kubectl get pod')

# Todo
- Add more usable feature (logging?)
- Document
- Test code

# Authors
- Tomofumi Hayashi (s1061123)
