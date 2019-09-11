package main

import (
	. "github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/spark-autoscaling/spark-deployment"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"os"
	"strconv"
)

func main() {
	// isInCluster is used to determine if this app is running in the developer's local side
	// or in the k8s cluster.
	isInCluster,err:=strconv.ParseBool(os.Getenv("IS_IN_CLUSTER"))
	if err!=nil{
		panic("Missing environment variable 'IS_IN_CLUSTER'")
	}
	cluster:=NewSparkCluster(isInCluster)

	// if cleanExistingDeployment is true, everything related to the Spark cluster will be redeployed,
	// including Spark master, master UI service, master service and all workers
	cleanExistingDeployment,err:=strconv.ParseBool(os.Getenv("CLEAN_EXISTING_DEPLOYMENT"))
	if err!=nil{
		panic("Missing environment variable 'CLEAN_EXISTING_DEPLOYMENT'")
	}
	cluster.Deploy(cleanExistingDeployment)
}