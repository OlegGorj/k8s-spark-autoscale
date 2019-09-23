package spark_deployment

import (
	"bytes"
	"github.com/google/uuid"
	"github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/k8s-util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestAutoScaling(t *testing.T)  {
	cluster:= NewSparkCluster(false)
	sparkMockClients:=NewSparkMockClients(cluster.sparkWorkerDeployment.deploymentClient,
		"zebinkang/cluster-autoscaling-test:0.1",
		cluster.sparkWorkerDeployment.sparkService,
		cluster.sparkWorkerDeployment.deploymentResource.Cores)
	sparkMockClients.deploymentClient.DeletePodWithLabel(sparkMockClients.labels)
	cleanExstingSparkCluster:=false
	go cluster.Deploy(cleanExstingSparkCluster)
	go periodCheckClientStatus(sparkMockClients)
	time.Sleep(15000 * time.Millisecond)

	for i:=0;i<3;i++{
		randomAddOrDelete(sparkMockClients)
		time.Sleep(10000*time.Millisecond)

	}
	sparkMockClients.deploymentClient.DeletePodWithLabel(sparkMockClients.labels)
	time.Sleep(10000*time.Millisecond)
	log.Println("Autoscaling testing is done")
}

func randomAddOrDelete(sparkMockClients *SparkMockClients)  {
	randomInt:=(time.Now().Nanosecond()/1000)%2

	pods:=sparkMockClients.deploymentClient.GetPodListWithLabels(sparkMockClients.labels)
	if randomInt==0 && len(pods)>0{
		log.Println("Rmove conn")
		sparkMockClients.deleteRandomSparkConn()
	} else {
		log.Println("Add conn")
		sparkMockClients.addWMockClient()
	}
}

func periodCheckClientStatus(sparkMockClients *SparkMockClients)   {
	for {
		pods:=sparkMockClients.deploymentClient.GetPodListWithLabels(sparkMockClients.labels)
		for _,pod:=range pods{
			if pod.Status.Phase=="Running" && pod.Status.ContainerStatuses[0].Ready {
				sparkMockClients.checkMockClientStatus(pod.Name)
			}
		}
		time.Sleep(time.Duration(sparkMockClients.statusCheckPeriod)* time.Millisecond)
	}
}

func (sparkMockClients SparkMockClients) checkMockClientStatus(mockClientPodName string)  {
	podsClient := sparkMockClients.deploymentClient.Clientset.CoreV1().Pods(sparkMockClients.deploymentClient.Namespace)
	req:=podsClient.GetLogs(mockClientPodName, &apiv1.PodLogOptions{
		TailLines: k8s_util.Int64Ptr(1),
	})
	readCloser, err := req.Stream()
	if err != nil {
		log.Println(err)
	} else {
		buf:=new(bytes.Buffer)
		buf.ReadFrom(readCloser)
		if err != nil {
			log.Println(err)
		}
		if strings.Contains(buf.String(),"worker lost"){
			log.Println("lost connection")
		}

		if strings.Contains(buf.String(),"SchedulerBackend is ready for scheduling beginning after reached minRegisteredResourcesRatio"){
			log.Println("Spark cluster does not scale out correclty")
		}
	}
}



type SparkMockClients struct {
	deploymentClient *k8s_util.DeploymentClient
	imageName string
	statusCheckPeriod int64
	labels map[string]string
	sparkPath string
	sparkService string
	coresPerClient string
}


func NewSparkMockClients(deploymentClient *k8s_util.DeploymentClient, imageName string, sparkService string, corePerWorker string) *SparkMockClients{
	sparkMockClients:=&SparkMockClients{
		deploymentClient: deploymentClient,
		imageName: imageName,
		labels:map[string]string{
			"spark": "mock-client",
		},
		sparkPath: "/usr/spark",
		sparkService: sparkService,
		coresPerClient: corePerWorker,

	}
	return sparkMockClients
}

func (sparkMockClients SparkMockClients) generateWorkerConfig() (*apiv1.Pod) {
	id, err:=uuid.NewUUID()
	if err !=nil {
		// handle error
	}
	sparkWorkerName:="spark-mock-client-"+id.String()
	sparkMockClientConfig := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: sparkWorkerName,
			Labels:map[string]string{
				"spark":"mock-client",
			},
		},
		Spec: apiv1.PodSpec{
			Containers: [] apiv1.Container {
				{
					Name: sparkWorkerName,
					Image: sparkMockClients.imageName,
					Command: []string{
						"/bin/sh",
						"-c",
					},
					Args: []string{
						//"echo $(hostname -i) "+sparkMasterDeployment.sparkMasterName+" >> /etc/hosts && python3 -m http.server",
						sparkMockClients.sparkPath+"/bin/spark-submit --master $SPARK_MASTER "+sparkMockClients.sparkPath+"/long_running_job.py",
					},

					Env: []apiv1.EnvVar{
						{
							Name: "SPARK_MASTER",
							Value: sparkMockClients.sparkService,
						},
						{
							Name:  "APP_NAME",
							Value: sparkWorkerName,
						},
						{
							Name:  "CORE_MAX",
							Value: "1",
						},
						{
							Name:  "EXECUTOR_CORES",
							Value: "1",
						},
						{
							Name:  "EXECUTOR_MEM",
							Value: "1g",
						},
					},
				},

			},
			ImagePullSecrets: []apiv1.LocalObjectReference{
				{"image-pull-secret-ibm-cloud"},
			},
		},
	}
	return sparkMockClientConfig
}

func (sparkMockClients SparkMockClients) addWMockClient() {
	_ = sparkMockClients.deploymentClient.AddPod(sparkMockClients.generateWorkerConfig())
}


func (sparkMockClients SparkMockClients) deleteRandomSparkConn() {
	pods:=sparkMockClients.deploymentClient.GetPodListWithLabels(sparkMockClients.labels)
	if len(pods)>0{
		i := rand.Intn(len(pods))
		sparkMockClients.deploymentClient.DeletePod(pods[i].Name)
	}
}