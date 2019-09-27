package cluster_controller

import (
	"fmt"
	k8sutil "github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/k8s-util"
	"gotest.tools/assert"
	apiv1 "k8s.io/api/core/v1"
	"time"

	//"gotest.tools/assert"
	"testing"
)

//Test in Sandbox env, need kubernetes sandbox yml
func TestGetPodListWithLabelsSandbox(t *testing.T) {
	scheduler := Scheduler{
		clientSet:  k8sutil.InitializeClient(false),
		workerPool: "kubernetes.io/hostname: 10.166.255.114",
		maxNode:    1,
	}
	nodeSelector := make(map[string]string)
	nodeSelector["pool"] = scheduler.workerPool
	podlists := scheduler.GetPodListWithLabels("spark-worker",nodeSelector)
	for _, pod := range podlists {
		fmt.Println(pod.Spec.NodeName)
	}
}

//Test in Dev env, need kubernetes dev yml
//func TestGetPodListWithLabelsDev(t *testing.T) {
//	scheduler := Scheduler{
//		clientSet: k8s_util.InitializeClient(false),
//		workerPool: "kubernetes.io/hostname: 10.166.255.114",
//		maxNode: 1,
//	}
//	nodeSelector := make(map[string]string)
//	nodeSelector["pool"] = scheduler.workerPool
//	podlists := scheduler.getPodListWithLabels("spark",nodeSelector)
//	for _, pod := range podlists {
//		fmt.Println(pod.Spec.NodeName)
//	}
//}

//test auto scale by time
func TestAutoScaleByTime(t *testing.T){
	calenderList := []AutoScalingCalender{
		{startHour:20,endHour:6,weekday:time.Monday,wholeDayOn:false},
		{startHour:20,endHour:22,weekday:time.Tuesday,wholeDayOn:false},
		{startHour:20,endHour:6,weekday:time.Wednesday,wholeDayOn:false},
		{startHour:10,endHour:18,weekday:time.Thursday,wholeDayOn:false},
		{startHour:20,endHour:6,weekday:time.Friday,wholeDayOn:false},
		//now weekends auto-scaling all on since why not
		{startHour:0,endHour:0,weekday:time.Saturday,wholeDayOn:true},
		{startHour:0,endHour:0,weekday:time.Sunday,wholeDayOn:true},
	}
	var autoscalingBool bool
	// Test valid pass midnight time , should be true
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,17,0, 0,0,0,time.UTC))
	assert.Assert(t,autoscalingBool)
	// Test valid turn on time at night , should be true
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,17,20, 0,0,0,time.UTC))
	assert.Assert(t,autoscalingBool)
	// Test valid turn on time between start and end, should be true
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,18,20, 1,0,0,time.UTC))
	assert.Assert(t,autoscalingBool)
	// Test turn off time between end and start, should be false
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,19,12, 0,0,0,time.UTC))
	assert.Assert(t,!autoscalingBool)
	// Test turn off time before start, should be false
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,20,8, 0,0,0,time.UTC))
	assert.Assert(t,!autoscalingBool)
	// Test turn off time after end, should be false
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,20,21, 0,0,0,time.UTC))
	assert.Assert(t,!autoscalingBool)
	// Test all true time , should be true
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,22,0, 0,0,0,time.UTC))
	assert.Assert(t,autoscalingBool)
	autoscalingBool = AutoScaleByTime(&calenderList,time.Date(2019,time.June,23,0, 0,0,0,time.UTC))
	assert.Assert(t,autoscalingBool)
}

func TestFindUnusedNodes(t *testing.T) {
	//p := fmt.Println
	podlists := []apiv1.Pod{
		{Spec:apiv1.PodSpec{NodeName:"12.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:"22.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:"32.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:"42.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:"52.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:""}},
	}
	nodelists := []string{
		"12.345.678.910",
		"32.345.678.910",
		"11.323.121.455",
		"23.532.353.245",
		"",
		"",
	}
	unused := FindUnusedNodes(podlists,nodelists)
	assert.Assert(t,len(unused) == 4)
	assert.Assert(t,unused[0]=="11.323.121.455")
	assert.Assert(t,unused[1]=="23.532.353.245")
	assert.Assert(t,unused[2]=="")
	assert.Assert(t,unused[3]=="")


	podlists = []apiv1.Pod{
		{Spec:apiv1.PodSpec{NodeName:"12.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:"22.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:""}},
		{Spec:apiv1.PodSpec{NodeName:""}},
	}
	nodelists = []string{
		"12.345.678.910",
		"22.345.678.910",
		"32.345.678.910",
		"11.323.121.455",
		"23.532.353.245",
	}
	unused = FindUnusedNodes(podlists,nodelists)
	assert.Assert(t,len(unused) == 3)
	assert.Assert(t,unused[0]=="32.345.678.910")
	assert.Assert(t,unused[0]=="11.323.121.455")
	assert.Assert(t,unused[0]=="23.532.353.245")
}

func TestFindPendingNodes(t *testing.T) {
	podlists := []apiv1.Pod{
		{Spec: apiv1.PodSpec{NodeName: "12.345.678.910"}},
		{Spec: apiv1.PodSpec{NodeName: "22.345.678.910"}},
		{Spec: apiv1.PodSpec{NodeName: ""}},
		{Spec: apiv1.PodSpec{NodeName: ""}},
	}
	count := FindPendingNodes(podlists)

	nodelists := []string{
		"12.345.678.910",
		"22.345.678.910",
		"32.345.678.910",
		"11.323.121.455",
		"23.532.353.245",
	}
	unused := FindUnusedNodes(podlists, nodelists)
	assert.Assert(t,len(unused)-count == 1)
}


func TestUnusedSize(t *testing.T) {
	podlists := []apiv1.Pod{
		{Spec:apiv1.PodSpec{NodeName:"12.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:"22.345.678.910"}},
		{Spec:apiv1.PodSpec{NodeName:""}},
		{Spec:apiv1.PodSpec{NodeName:""}},
	}
	count := FindPendingNodes(podlists)
	assert.Assert(t,count == 2)
}

func TestSparkAlgoV1(t *testing.T) {
	// Test invalid inputs: minNode > maxNode
	_,_,err := SparkAlgoV1(0,0,0,5,4)
	assert.Assert(t,err != nil)
	// Test invalid inputs: unusedNodes > totalNodes
	_,_,err = SparkAlgoV1(1,2,0,2,4)
	assert.Assert(t,err != nil)
	// Test invalid inputs: min+ extra > max
	_,_,err = SparkAlgoV1(1,2,3,3,5)
	assert.Assert(t,err != nil)
	// Number of Nodes < min+ extra, should scale up
	scaleUp,scaleDown,err := SparkAlgoV1(0,0,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,scaleUp && !scaleDown)
	// Number of Nodes = min + extra, has no idle nodes, should scale up
	scaleUp,scaleDown,err = SparkAlgoV1(3,0,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && scaleUp)
	// Number of Nodes = min + extra, has one idle node, should stay the same
	scaleUp,scaleDown,err = SparkAlgoV1(3,1,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && !scaleUp)
	// Number of Nodes > min + extra, but no idle node, should scale up
	scaleUp,scaleDown,err = SparkAlgoV1(5,0,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && scaleUp)
	// Number of Nodes > min + extra, has idle node = extra, should stay the same
	scaleUp,scaleDown,err = SparkAlgoV1(7,1,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && !scaleUp)
	// Number of Nodes = max node, has no idle node, but should stay the same
	scaleUp,scaleDown,err = SparkAlgoV1(9,0,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && !scaleUp)
	// Number of Nodes is between min + extra to max, has idle node > extra, should drop a node
	scaleUp,scaleDown,err = SparkAlgoV1(7,2,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,scaleDown && !scaleUp)
	// Number of Nodes = max node , no idle node, should not drop , duplicated test case
	scaleUp,scaleDown,err = SparkAlgoV1(9,0,2,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && !scaleUp)
	// Number of Nodes = min + extra, has more idle nodes than extra, but should not drop
	scaleUp,scaleDown,err = SparkAlgoV1(3,3,1,2,9)
	assert.Assert(t,err == nil)
	assert.Assert(t,!scaleDown && !scaleUp)
}

//func TestRefreshToken(t *testing.T) {
//	scheduler:= Scheduler{
//		clusterClient:&IBMCloudClient{
//			lastRefreshedTime: time.Date(2019,time.June,24,20, 0,0,0,time.UTC),
//			},
//	}
//	scheduler.RefreshToken(time.Date(2019,time.June,24,19,0,0,0,time.UTC))
//	assert.Assert(t,scheduler.clusterClient.lastRefreshedTime ==time.Date(2019,time.June,24,20,0,0,0,time.UTC))
//
//	scheduler.RefreshToken(time.Date(2019,time.June,24,20,50,0,0,time.UTC))
//	assert.Assert(t,scheduler.clusterClient.lastRefreshedTime ==time.Date(2019,time.June,24,20,0,0,0,time.UTC))
//
//	scheduler.RefreshToken(time.Date(2019,time.June,24,20,59,0,0,time.UTC))
//	assert.Assert(t,scheduler.clusterClient.lastRefreshedTime ==time.Date(2019,time.June,24,20,0,0,0,time.UTC))
//
//	scheduler.RefreshToken(time.Date(2019,time.June,24,21,59,0,0,time.UTC))
//	assert.Assert(t,scheduler.clusterClient.lastRefreshedTime ==time.Date(2019,time.June,24,20,0,0,0,time.UTC))
//
//}