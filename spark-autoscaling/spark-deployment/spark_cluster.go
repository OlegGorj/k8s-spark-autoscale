package spark_deployment

import (
	"github.com/json-iterator/go"
	"github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/k8s-util"
	"log"
	"os"
	"strconv"
	"time"
)

/**
This struct contains data for a Spark cluster
 */
type SparkCluster struct {
	sparkMasterDeployment *SparkMasterDeployment
	sparkWorkerDeployment *SparkWorkerDeployment
}

/**
Constructor for SparkCluster struct
 */
func NewSparkCluster(inClusterDeployment bool) *SparkCluster{
	sparkDeploymentClient:= k8s_util.NewDeploymentClient(inClusterDeployment,os.Getenv("SPARK_CLUSTER_NAMESPACE"),)
	sparkWorkerDeploymentResource:= k8s_util.NewDeploymentResource(
		os.Getenv("SPARK_WORKER_CORES"),
		os.Getenv("SPARK_WORKER_MEM"),
		os.Getenv("SPARK_WORKER_CONTAINER_CPU"),
		os.Getenv("SPARK_WORKER_CONTAINER_MEM"))
	sparkMasterDeploymentResource:=k8s_util.NewDeploymentResource(
		os.Getenv("SPARK_MASTER_CORES"),
		os.Getenv("SPARK_MASTER_MEM"),
		os.Getenv("SPARK_MASTER_CONTAINER_CPU"),
		os.Getenv("SPARK_MASTER_CONTAINER_MEM"))
	sparkMasterDeployment:= NewSparkMasterDeployment(
		sparkDeploymentClient,
		os.Getenv("SPARK_MASTER_IMAGE"),
		sparkMasterDeploymentResource)
	sparkWorkerDeployment:= NewSparkWorkerDeployment(
		//TODO: check core max does not work properly problem
		sparkDeploymentClient,
		os.Getenv("SPARK_WORKER_IMAGE"),
		sparkWorkerDeploymentResource,
		os.Getenv("SPARK_WORKER_OPTS"))
	return &SparkCluster{
		sparkMasterDeployment:sparkMasterDeployment,
		sparkWorkerDeployment:sparkWorkerDeployment,
	}
}


/**
This function is used to deploy spark cluster and start autoscaling
 */
func (sparkCluster SparkCluster) Deploy(cleanExisting bool) {
	if cleanExisting {
		sparkCluster.sparkWorkerDeployment.removeAllWorker()
		sparkCluster.sparkMasterDeployment.Deploy()
		sparkCluster.autoScale()
	} else{
		sparkCluster.autoScale()
	}
}

/**
This function is to auto scale the Spark cluster with keep scaling in or out the cluster through
tracking the utilization of workers
 */
func (sparkCluster SparkCluster) autoScale()  {
	for {
		clusterInfo,err:=sparkCluster.sparkWorkerDeployment.getClusterInfo()
		if err != nil {
			log.Println(err)
		}else{
			// count cores in use based on the information from Spark master json
			coresused:=jsoniter.Get(clusterInfo, "coresused").ToInt()
			coresPerWorker, _ :=strconv.Atoi(sparkCluster.sparkWorkerDeployment.deploymentResource.Cores)
			targetCores:=coresused+coresPerWorker*sparkCluster.sparkWorkerDeployment.extraSparkWorker
			// count spark worker num based on nums of spark worker pods (including the pending ones)
			hasError := false
			currWorkerNum:= len(sparkCluster.sparkWorkerDeployment.getWorkers(&hasError))
			if hasError {time.Sleep(1000*time.Millisecond);continue}
			cores:=currWorkerNum*coresPerWorker
			log.Println("target cores:",targetCores)
			log.Println("current cores:",cores)
			if cores<targetCores{
				sparkCluster.scaleOut()
			}
			if cores>targetCores {
				sparkCluster.scaleIn()
			}
		}
		time.Sleep(1000*time.Millisecond)
	}
}

/**
This function is to scale out the Spark cluster by adding a new worker to the cluster
 */
func (sparkCluster SparkCluster) scaleOut()  {
	err := sparkCluster.sparkWorkerDeployment.addWorker()
	if err != nil {
		log.Println(err)
	}
}

/**
This function is to scale in the Spark cluster by deleting an random idle worker
 */
func (sparkCluster SparkCluster) scaleIn()  {
	podToRemove:=sparkCluster.sparkWorkerDeployment.podToRemove()
	err := sparkCluster.sparkWorkerDeployment.removeWorker(podToRemove)
	if err != nil {
		log.Println(err)
	}
}