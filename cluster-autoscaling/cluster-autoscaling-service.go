package main

import (
	. "github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/cluster-autoscaling/cluster-controller"
	k8sutil "github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/k8s-util"
	"os"
	"strconv"
)
func main() {
	isInCluster,err:=strconv.ParseBool(os.Getenv("IS_IN_CLUSTER"))
	if err!=nil{
		panic("Missing environment variable 'IS_IN_CLUSTER'")
	}
	ignoreSchedule,err:=strconv.ParseBool(os.Getenv("IGNORE_SCHEDULE"))	// whether ignore auto-scaling schedule and force auto scaling to be on
	if err!=nil{
		panic("Missing environment variable 'IGNORE_SCHEDULE'")
	}
	workerPool := os.Getenv("WORKER_POOL_NAME")
	nameSpace := os.Getenv("NAMESPACE")
	maxNode, _ := strconv.Atoi(os.Getenv("MAX_NODE"))
	minNode, _ := strconv.Atoi(os.Getenv("MIN_NODE"))
	extraNode, _ := strconv.Atoi(os.Getenv("EXTRA_NODE"))
	// now spark worker only
	ibmCloudClient := NewIBMCloudClient()
	k8sClient := k8sutil.InitializeClient(isInCluster)
	sparkScheduler := NewScheduler(ibmCloudClient,k8sClient,workerPool,nameSpace,maxNode,minNode,extraNode)
	sparkScheduler.AutoScale(ignoreSchedule)
}