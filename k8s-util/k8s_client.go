package k8s_util

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

/*
This function is used to initialize a k8s clientset. If the user is trying to
use this function from their local side, set inCluster as false; if this function
is used by an app running in the k8s pod, set inCluster as true
 */
func InitializeClient(inCluster bool) (* kubernetes.Clientset){
	if inCluster{
		return initializeInClusterClient()
	} else {
		return initializeOutOfClusterClient()
	}
}

func initializeInClusterClient() (* kubernetes.Clientset) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	} else {
		return clientset
	}
}

/*
Initialization of the clientset out ofq
the k8s cluster requires set KUBECONFIG_ABSOLUTE_PATH in
the environment
 */
func initializeOutOfClusterClient() (* kubernetes.Clientset) {
	var kubeconfig *string
	kubeconfigAbsolutePath:=os.Getenv("KUBECONFIG_ABSOLUTE_PATH")
	kubeconfig = flag.String("kubeconfig",kubeconfigAbsolutePath, "(optional) absolute path to the kubeconfig file")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	} else {
		return clientset
	}
}