package main
// k8sclient base code is comes from github.com/intel/multus-cni.

import (
	//"flag"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
/*
	"k8s.io/client-go/util/retry"
*/
	"os"
)

// NoK8sNetworkError indicates error, no network in kubernetes
type NoK8sNetworkError struct {
	message string
}

type KubeClient interface {
	GetRawWithPath(path string) ([]byte, error)
	GetPod(namespace, name string) (*v1.Pod, error)
	UpdatePodStatus(pod *v1.Pod) (*v1.Pod, error)
	GetNode(name string) (*v1.Node, error)
	List() (*v1.NodeList, error)
}

type clientInfo struct {
	Client       KubeClient
	Podnamespace string
	Podname      string
}

func (e *NoK8sNetworkError) Error() string { return string(e.message) }

type defaultKubeClient struct {
	client kubernetes.Interface
}

// defaultKubeClient implements KubeClient
var _ KubeClient = &defaultKubeClient{}

func (d *defaultKubeClient) GetRawWithPath(path string) ([]byte, error) {
	return d.client.ExtensionsV1beta1().RESTClient().Get().AbsPath(path).DoRaw()
}

func (d *defaultKubeClient) GetPod(namespace, name string) (*v1.Pod, error) {
	return d.client.Core().Pods(namespace).Get(name, metav1.GetOptions{})
}

func (d *defaultKubeClient) UpdatePodStatus(pod *v1.Pod) (*v1.Pod, error) {
	return d.client.Core().Pods(pod.Namespace).UpdateStatus(pod)
}

func (d *defaultKubeClient) GetNode(name string) (*v1.Node, error) {
	return d.client.Core().Nodes().Get(name, metav1.GetOptions{})
}

func (d *defaultKubeClient) List() (*v1.NodeList, error) {
	return d.client.Core().Nodes().List(metav1.ListOptions{})
}

func GetK8sClient(kubeconfig string, kubeClient KubeClient) (KubeClient, error) {
	// If we get a valid kubeClient (eg from testcases) just return that
	// one.
	if kubeClient != nil {
		return kubeClient, nil
	}

	var err error
	var config *rest.Config

	// Otherwise try to create a kubeClient from a given kubeConfig
	if kubeconfig != "" {
		// uses the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("GetK8sClient: failed to get context for the kubeconfig %v, refer Multus README.md for the usage guide: %v", kubeconfig, err)
		}
	} else if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		// Try in-cluster config where multus might be running in a kubernetes pod
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("createK8sClient: failed to get context for in-cluster kube config, refer Multus README.md for the usage guide: %v", err)
		}
	} else {
		// No kubernetes config; assume we shouldn't talk to Kube at all
		return nil, nil
	}

	// creates the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &defaultKubeClient{client: client}, nil
}


func GetHostIP (nodeaddr *[]v1.NodeAddress) (hostname, hostip string) {
	for _, val := range *nodeaddr {
		switch val.Type {
		case v1.NodeHostName:
			hostname = val.Address
		case v1.NodeInternalIP:
			hostip = val.Address
		}
	}
	return
}

/*
var kubeconfig = flag.String("kubeconfig", "/etc/kubeconfig", "help message for s option")
func main() {
	flag.Parse()
	kubeClient, err := GetK8sClient(*kubeconfig, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err:%v", err)
	}
	pod, err := kubeClient.GetPod("default", "centos")
	if err != nil {
		fmt.Fprintf(os.Stderr, "err:%v", err)
	}
	for _, val := range pod.Status.ContainerStatuses {
		fmt.Printf("name: %s: %s\n", val.Name, val.ContainerID)
	}
	fmt.Printf("Host IP: %s\n", pod.Status.HostIP)
}
*/
