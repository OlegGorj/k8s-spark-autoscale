package k8s_util

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"log"
	"time"
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

type DeploymentClient struct {
	Clientset *kubernetes.Clientset
	Namespace string
}

func NewDeploymentClient(inCluster bool, namespace string) *DeploymentClient {
	return &DeploymentClient{
		Clientset: InitializeClient(inCluster),
		Namespace: namespace,
	}
}
/*
Create a Kubernetes Deployment given a deployment config
 */
func (deploymentClient *DeploymentClient) CreateDeployment(deploymentConfig * appsv1.Deployment){
	deploymentsClient := deploymentClient.Clientset.AppsV1().Deployments(deploymentClient.Namespace)


	log.Println("Creating deployment...")
	result, err := deploymentsClient.Create(deploymentConfig)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
}
/*
Delete a Kubernetes Deployment given the name of that deployment
the function keeps tracking the result until it is completed
 */
func (deploymentClient *DeploymentClient) DeleteDeployment(deploymentName string){
	log.Println("Deleting deployment ", deploymentName)
	deploymentsClient := deploymentClient.Clientset.AppsV1().Deployments(deploymentClient.Namespace)
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(deploymentName, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		log.Println(err)
	}
	for {
		result, _ :=deploymentsClient.Get(deploymentName,metav1.GetOptions{})
		if result.ObjectMeta.Name=="" {
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}
	log.Println("Deleted deployment ", deploymentName)
}

/*
Return a list of pods given a label map, if there is an error, it will return an empty list
 */
func (deploymentClient DeploymentClient) GetPodListWithLabels(targetLabels map[string]string) ([] apiv1.Pod,error) {
	podsClient := deploymentClient.Clientset.CoreV1().Pods(deploymentClient.Namespace)
	set:=labels.Set(targetLabels)
	pods, err := podsClient.List(metav1.ListOptions{
			LabelSelector: set.AsSelector().String(),
		})
	/*
	TODO:  we need to handle the case where Get an error and pod is nil,
	TODO: so total worker size is 0, and results in the spark worker being scaled down
	 */
	if err != nil {
		log.Println("Error in getting pods with labels: ",err)
		return [] apiv1.Pod{},err
	}
	if pods == nil{
		return [] apiv1.Pod{},err
	}
	return pods.Items,nil
}

/*
Given a pod config , create a pod,
if succeed, the pod name is returned,
otherwise, return an empty string
 */
func (deploymentClient DeploymentClient) AddPod(podConfig *apiv1.Pod) string{
	podsClient := deploymentClient.Clientset.CoreV1().Pods(deploymentClient.Namespace)

	log.Println("Creating pod "+podConfig.Name+"...")
	_, err := podsClient.Create(podConfig)
	if err != nil{
		return ""
	}
	for {
		result, _ :=podsClient.Get(podConfig.Name,metav1.GetOptions{})
		if result.ObjectMeta.Name!="" {
			return result.ObjectMeta.Name
		}
		time.Sleep(1000 * time.Millisecond)

	}
}

/*
Delete a pod with podname
if failed, will log the error message
 */
func (deploymentClient *DeploymentClient) DeletePod(podName string){
	log.Println("Deleting Pod "+podName+"...")
	podsClient := deploymentClient.Clientset.CoreV1().Pods(deploymentClient.Namespace)
	deletePolicy := metav1.DeletePropagationForeground
	err := podsClient.Delete(podName, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		log.Println("Delete Pod error: ",err)
	}else {
		log.Println("Deleted Pod : ",podName)
	}
}

/*
Delete all the pods with same labels defined in input
 */
func (deploymentClient *DeploymentClient) DeletePodWithLabel(targetLabels map[string]string){
	log.Println("Deleting Pods ...")
	pods,err:=deploymentClient.GetPodListWithLabels(targetLabels)
	if err != nil {
		log.Println("Cannot delete pods")
		return
	}
	for _, p := range pods {
		deploymentClient.DeletePod(p.Name)
	}
	log.Println("Deleted Pods.")
}

/*
Generate the configuration for go-client to create a service
Input
-----
serviceName:	the name of the service
servicePort:	the port of the service
serviceLabels:	this will be used to label both the object metadata and selector
 */
func generateServiceConfig(serviceName string,servicePort int32, serviceLabels map[string]string)(* apiv1.Service){
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			Labels: serviceLabels,
		},
		Spec: apiv1.ServiceSpec{
			Selector: serviceLabels,	//used to match the target deployment or pods
			Ports: []apiv1.ServicePort{
				{
					Name:          serviceName,
					Protocol:      apiv1.ProtocolTCP,
					Port: servicePort,
					TargetPort: intstr.IntOrString{
						Type: intstr.Int,
						IntVal: servicePort,
						StrVal: string(servicePort),
					},
				},
			},
		},
	}
	return service
}


func (deploymentClient *DeploymentClient) createSecret(secretName string,secretUID string) {
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
			Namespace: deploymentClient.Namespace,
			SelfLink: "/api/v1/namespaces/"+deploymentClient.Namespace +"/secrets/image-pull-secret-ibm-cloud",
			UID: types.UID("de9e018b-4a5e-11e9-8220-022f415d61d9"),
			ResourceVersion: "7337242",
		},
		Type: "kubernetes.io/dockerconfigjson",
	}

	log.Println("Creating Secret "+secretName)
	_, err := deploymentClient.Clientset.CoreV1().Secrets(deploymentClient.Namespace).Create(secret)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Created Secret "+secretName)
}

/*
Create a k8s service
note: serviceLabels is used for service selector
 */
func (deploymentClient *DeploymentClient) CreateService(serviceName string,servicePort int32, serviceLabels map[string]string){
	log.Println("Creating Service "+serviceName)
	_, err := deploymentClient.Clientset.CoreV1().Services(deploymentClient.Namespace).Create(
		generateServiceConfig(serviceName, servicePort, serviceLabels))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Created Service "+serviceName)
}

func (deploymentClient *DeploymentClient) DeleteService(serviceName string){
	deletePolicy := metav1.DeletePropagationForeground
	log.Println("Deleting Service "+serviceName)
	if err := deploymentClient.Clientset.CoreV1().Services(deploymentClient.Namespace).Delete(serviceName, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		log.Println(err)
	} else {
		log.Println("Deleted Service "+serviceName)
	}
}

