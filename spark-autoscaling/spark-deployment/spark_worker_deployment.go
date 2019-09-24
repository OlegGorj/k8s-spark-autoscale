package spark_deployment

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/google/uuid"
	"github.com/json-iterator/go"
	"github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/k8s-util"
	"io/ioutil"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const NodePending = "Pending"

/**
SparkWorkerDeployment struct contains data related to spark workers
 */
type SparkWorkerDeployment struct{
	deploymentClient     *k8s_util.DeploymentClient
	imageName            string
	labels               map[string]string
	nodeSelector         map[string]string
	sparkWorkerOpts      string
	sparkService         string
	sparkPath            string
	sparkMasterWebuiPort string
	workerNameToNet      map[string]string
	deploymentResource   *k8s_util.DeploymentResource
	extraSparkWorker     int
}

/**
Constructor for SparkWorkerDeployment struct
 */
func NewSparkWorkerDeployment(deploymentClient *k8s_util.DeploymentClient,
	imageName string,
	resource *k8s_util.DeploymentResource,
	sparkWorkerOpts string) *SparkWorkerDeployment{
	extraSparkWorker, _ :=strconv.Atoi(os.Getenv("EXTRA_SPARK_WORKER"))
	sparkWorker:=&SparkWorkerDeployment{
		deploymentClient: deploymentClient,
		imageName:        imageName,
		labels:map[string]string{
			"component": "spark-worker",
			"pool": os.Getenv("SPARK_WORKER_POOL"),
		},
		nodeSelector:map[string]string{
			"pool": os.Getenv("SPARK_WORKER_POOL"),
		},
		sparkWorkerOpts: sparkWorkerOpts,
		sparkPath: "/usr/spark",
		sparkService: "spark://spark-master:7077",
		sparkMasterWebuiPort: "8080",
		workerNameToNet:map[string]string{},
		deploymentResource: resource,
		extraSparkWorker: extraSparkWorker,
	}
	sparkWorker.prepareWorkerInfo()
	return sparkWorker
}

/**
This function is to generate a map between node network information and pod name for each worker.
 */
func (sparkWorkerDeployment SparkWorkerDeployment)  prepareWorkerInfo (){
	hasError := false
	pods := sparkWorkerDeployment.getWorkers(&hasError)
	if hasError {return}
	for _,pod := range pods{
		if pod.Status.Phase=="Running"{
			// note: even though the pod is running, it still takes time for this worker to be shown
			// in spark master
			workerNet:=sparkWorkerDeployment.findWorkerNetWithPodName(pod.Name)
			sparkWorkerDeployment.workerNameToNet[pod.Name]=workerNet
		}
		if pod.Status.Phase=="Pending"{
			sparkWorkerDeployment.workerNameToNet[pod.Name]=""
		}
	}
}

/**
This function is to generate configuration for a Spark worker pod
 */
func (sparkWorkerDeployment SparkWorkerDeployment) generateWorkerConfig() (*apiv1.Pod) {
	id, err:=uuid.NewUUID()
	if err !=nil {
		// handle error
	}
	sparkWorkerName:="spark-worker-"+id.String()
	sparkWorkerConfig := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: sparkWorkerName,
			Labels:sparkWorkerDeployment.labels,
		},
		Spec: apiv1.PodSpec{
			Containers: [] apiv1.Container {
				{
					Name: sparkWorkerName,
					Image: sparkWorkerDeployment.imageName,
					Command: []string{
						"/bin/sh",
						"-c",
					},
					Args: []string{
						//"echo $(hostname -i) "+sparkMasterDeployment.sparkMasterName+" >> /etc/hosts && python3 -m http.server",
						sparkWorkerDeployment.sparkPath+"/bin/spark-class org.apache.spark.deploy.worker.Worker "+sparkWorkerDeployment.sparkService,
					},
					Ports: []apiv1.ContainerPort{
						{
							Name:          "worker",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: 8081,
						},
					},
					Env: []apiv1.EnvVar{
						{
							Name: "SPARK_DAEMON_MEMORY",
							Value: "1g",
						},
						{
							Name:  "SPARK_WORKER_CORES",
							Value: sparkWorkerDeployment.deploymentResource.Cores,
						},
						{
							Name:  "SPARK_WORKER_MEMORY",
							Value: sparkWorkerDeployment.deploymentResource.Mem,
						},
						{
							Name:  "SPARK_WORKER_OPTS",
							Value: sparkWorkerDeployment.sparkWorkerOpts,
						},
					},
					Resources: sparkWorkerDeployment.deploymentResource.GenerateResourceRequirements(),
				},

			},
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{"image-pull-secret-ibm-cloud"},
			},
			NodeSelector: sparkWorkerDeployment.nodeSelector,
		},
	}
	return sparkWorkerConfig

}

/**
This function retrieves Spark cluster from Spark master in json formation through http
 */
func (sparkWorkerDeployment SparkWorkerDeployment) getClusterInfo() ([]byte, error) {
	sparkClusterInfoURL:=os.Getenv("SPARK_CLUSTER_INFO_URL")
	response, err := http.Get(sparkClusterInfoURL)
	if err!=nil{
		log.Println("The cluster is down")
		return nil,err
	}
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return nil,err
	}
	return responseData,nil
}

/**
This function returns the pod name of a idle worker randomly. If there is no idle workers in
the cluster, it returns empty string.
 */
func (sparkWorkerDeployment SparkWorkerDeployment) podToRemove() string {
	hasError := false
	pods:=sparkWorkerDeployment.getWorkers(&hasError)
	if hasError {return ""}
	for _, pod := range pods{
		if pod.Status.Phase == "Pending"{
			return pod.Name
		}
	}
	clusterInfo,err:=sparkWorkerDeployment.getClusterInfo()
	if err != nil{
		log.Println(err)
		return ""
	}
	for i:=0;i<len(strings.Split(jsoniter.Get(clusterInfo, "workers").ToString(),"},"));i++{
		workerInfo:=jsoniter.Get(clusterInfo, "workers",i).ToString()
		if jsoniter.Get([]byte(workerInfo), "coresused",).ToString() == "0" && jsoniter.Get([]byte(workerInfo), "state",).ToString()=="ALIVE" {
			workerID := jsoniter.Get([]byte(workerInfo), "id", ).ToString()
			podIP:=strings.Split(workerID,"-")[2]
			podPort:=strings.Split(workerID,"-")[3]
			podNet:=string(podIP+":"+podPort)
			podName:=""
			for name, net := range sparkWorkerDeployment.workerNameToNet {
				if net==podNet{
					podName=name
					break
				}
			}
			return podName
		}
	}
	return ""
}

/**
This function returns the ALIVE workers number in the Spark cluster
 */
func (sparkWorkerDeployment SparkWorkerDeployment) getWorkerNumFromMaster() (int,error) {
	clusterInfo,err:=sparkWorkerDeployment.getClusterInfo()
	if err!=nil{
		return -1,err
	}
	workerCount:=0
	for i:=0;i<len(strings.Split(jsoniter.Get(clusterInfo, "workers").ToString(),"},"));i++{
		workerInfo:=jsoniter.Get(clusterInfo, "workers",i).ToString()
		if jsoniter.Get([]byte(workerInfo), "state",).ToString()=="ALIVE" {
			workerCount+=1
		}
	}
	return workerCount,nil
}


func (sparkWorkerDeployment SparkWorkerDeployment) getWorkers(hasError *bool)  [] apiv1.Pod {
	workers,err:=sparkWorkerDeployment.deploymentClient.GetPodListWithLabels(sparkWorkerDeployment.labels)
	if err != nil {*hasError=true}
	for _, worker:=range workers{
		if sparkWorkerDeployment.workerNameToNet[worker.Name]==NodePending{
			if worker.Status.Phase=="Running"{
				workerNet:=sparkWorkerDeployment.findWorkerNetWithPodName(worker.Name)
				sparkWorkerDeployment.workerNameToNet[worker.Name]=workerNet
			}
		}

	}
	return workers
}

/**
This function is to add a Spark worker to Spark cluster, then keep tracking the status of adding until
the new worker joins the cluster
 */
func (sparkWorkerDeployment SparkWorkerDeployment) addWorker() error {
	hasError := false
	currWorkerNum:= len(sparkWorkerDeployment.getWorkers(&hasError))
	if hasError {return errors.New("failed to get pods information")}
	newWorkerName:=sparkWorkerDeployment.deploymentClient.AddPod(sparkWorkerDeployment.generateWorkerConfig())
	for {
		hasError = false
		updatedWorkerNum:=len(sparkWorkerDeployment.getWorkers(&hasError))
		if hasError {return errors.New("failed to get pods information")}
		if updatedWorkerNum>currWorkerNum {
			sparkWorkerDeployment.workerNameToNet[newWorkerName]=NodePending
			break
		}
		time.Sleep(1000*time.Millisecond)
	}
	return nil
}

/**
This function is to get a worker's pod name based on the worker's net information (FORMAT: "NODE_IP:NODE_PORT")
 */
func (sparkWorkerDeployment SparkWorkerDeployment) findWorkerNetWithPodName(podName string) string {
	podsClient := sparkWorkerDeployment.deploymentClient.Clientset.CoreV1().Pods(sparkWorkerDeployment.deploymentClient.Namespace)
	req:=podsClient.GetLogs(podName, &apiv1.PodLogOptions{})
	readCloser, err := req.Stream()
	if err != nil {
		log.Println(err)
	}
	buf:=new(bytes.Buffer)
	_,_ = buf.ReadFrom(readCloser)
	workerNet:=""
	if err != nil {
		log.Println(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(buf.String()))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(),"Starting Spark worker"){
			workerNet=strings.Split(scanner.Text()," ")[7]
		}
	}
	return workerNet
}

/**
This function is to remove a worker in the Spark cluster based on the pod name of that worker,
then keep tracking the status of that worker until the worker is removed from Spark cluster
 */
func (sparkWorkerDeployment SparkWorkerDeployment) removeWorker(podName string) error {
	if podName!=""{
		hasError := false
		workersNum:=len(sparkWorkerDeployment.getWorkers(&hasError))
		if hasError {return errors.New("failed to get pods information")}
			sparkWorkerDeployment.deploymentClient.DeletePod(podName)
		for {
			hasError = false
			updatedWorkersNum:=len(sparkWorkerDeployment.getWorkers(&hasError))
			if hasError {return errors.New("failed to get pods information")}
			if workersNum>updatedWorkersNum {
				delete(sparkWorkerDeployment.workerNameToNet,podName)
				break
			}
			time.Sleep(1000*time.Millisecond)
		}
	}
	return nil
}

/**
This function is to delete all workers in Spark cluster
 */
func (sparkWorkerDeployment SparkWorkerDeployment) removeAllWorker()  {
	sparkWorkerDeployment.deploymentClient.DeletePodWithLabel(sparkWorkerDeployment.labels)
}
