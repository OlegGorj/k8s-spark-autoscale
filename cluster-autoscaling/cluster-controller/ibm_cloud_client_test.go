package cluster_controller

import (
	"gotest.tools/assert"
	"reflect"
	"testing"
)

func TestClusterResourceGroup(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	clusterResourceGroup:=cloudClient.getClusterResourceGroup()
	assert.Assert(t,reflect.TypeOf(clusterResourceGroup).String()=="string")
}

func TestGetWorkerPools(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	workerPoolNames:=cloudClient.GetWorkerPools()
	assert.Assert(t,len(workerPoolNames)>0)
}


func TestGetWorkerNodesIP(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	workerPoolNodesIP:=cloudClient.getWorkersNodesIP("default")
	assert.Assert(t,len(workerPoolNodesIP)>0)
}

func TestGetWorkersID(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	workerPoolNodesIP:=cloudClient.getWorkersID("spark-worker", "10.166.255.119")
	assert.Assert(t,len(workerPoolNodesIP)>0)
}


func TestRemoveWorker(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	successfulToRemoveWorker:=cloudClient.removeWorker("spark-worker", "10.166.255.73")
	assert.Assert(t,successfulToRemoveWorker==true)
}


func TestAddOneWorker(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	successfulToAddWorker:=cloudClient.addOneWorker("spark-worker")
	assert.Assert(t,successfulToAddWorker==true)
}

/*
Be careful when running it since it will label the specified worker pool
 */
func TestLabelWorkerPool(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	successfulToAddWorker:=cloudClient.labelWorkerPool("jhub-user", "pool", "jhub-user")
	assert.Assert(t,successfulToAddWorker==true)
}

func TestRefreshToken(t *testing.T) {
	var cloudClient= NewIBMCloudClient()
	cloudClient.iamToken = "wrong_token"
	prevToken := cloudClient.iamToken
	cloudClient.RefreshToken()
	assert.Assert(t,cloudClient.iamToken != "")
	assert.Assert(t,cloudClient.iamToken != prevToken)
}