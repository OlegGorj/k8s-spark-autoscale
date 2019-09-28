package cluster_controller

import (
	"errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

type Scheduler struct {
	clusterClient	*IBMCloudClient
	clientSet 		*kubernetes.Clientset
	workerPool 		string		//name of the workerPool
	nameSpace		string		//name of the namespace
	maxNode			int		//maximum nodes the workerPool is allowed to own
	minNode			int 	//minimum nodes the workerPool is allowed to own
	extraNode		int 	//extra idle nodes for additional usage
	timeInterval	time.Duration		//time interval in SECONDS to check auto scaling
}

func NewScheduler(ibmCloudClient *IBMCloudClient,k8ClientSet *kubernetes.Clientset,
	workerPoolName string,nameSpace string, maxNodeNum int,minNodeNum int,extraNode int) *Scheduler {
	return &Scheduler{
		clusterClient:	ibmCloudClient,
		clientSet: 		k8ClientSet,
		workerPool:		workerPoolName,
		nameSpace:		nameSpace,
		maxNode:		maxNodeNum,
		minNode:		minNodeNum,
		extraNode:		extraNode,
		timeInterval:	15,
	}
}

// Structure for auto scaling calender, it manages the start and end time for each weekday,
// for simplicity now, only one start hour and end hour are allowed to define
type AutoScalingCalender struct {
	startHour	int	//when auto scaling turns on
	endHour		int	//when auto scaling turns off
	weekday		time.Weekday //the weekday for this structure
	wholeDayOn	bool //the entire day is on
}

// Initialize the calender when auto scaling being instantiated,
// This is for husky practice and it is very project dependent
func InitAutoScalingCalender()*[]AutoScalingCalender{
	calenderList := []AutoScalingCalender{
		// week day from 8pm to 6am auto scaling on
		{startHour:20,endHour:6,weekday:time.Monday,wholeDayOn:false},
		{startHour:20,endHour:6,weekday:time.Tuesday,wholeDayOn:false},
		{startHour:20,endHour:6,weekday:time.Wednesday,wholeDayOn:false},
		{startHour:20,endHour:6,weekday:time.Thursday,wholeDayOn:false},
		{startHour:20,endHour:6,weekday:time.Friday,wholeDayOn:false},
		//now weekends auto-scaling all on since why not
		{startHour:0,endHour:0,weekday:time.Saturday,wholeDayOn:true},
		{startHour:0,endHour:0,weekday:time.Sunday,wholeDayOn:true},
	}
	return &calenderList
}

/*
Evaluate the workerNodes/Pods usage condition in a worker pool and make the decision of adding/deleting worker nodes
Current Algorithm is simple:
	Checking all the Nodes and Pods in the workerPool, get the list of nodes that are not working, and drop one randomly,
	further optimization will be made after version 1.0 is up and stable
 */
func (schedulerClient *Scheduler) AutoScale(ignoreTimeSchedule bool){
	calender := InitAutoScalingCalender()
	timeZone := os.Getenv("TIME_ZONE")
	if timeZone == ""{
		timeZone = "America/Edmonton"
	}
	loc ,_ := time.LoadLocation(timeZone)
	log.Println("Time Zone is set to ",loc.String())
	for {
		// Check if auto scaling on
		if AutoScaleByTime(calender,time.Now().In(loc)) || ignoreTimeSchedule {
			// Get the list of nodes in	the workerPool
			nodesList := schedulerClient.clusterClient.getWorkersNodesIP(schedulerClient.workerPool)
			// Get the list of pods with matching node selector
			nodeSelector := make(map[string]string)
			nodeSelector["pool"] = schedulerClient.workerPool
			podsList := schedulerClient.GetPodListWithLabels(schedulerClient.nameSpace,nodeSelector)
			// podList nil means there is error getting the pod
			if podsList == nil {
				log.Println("Can't get pod list in the workerPool, skip this round")
				time.Sleep(schedulerClient.timeInterval * time.Second) //check after the interval
				continue
			}
			// It is a bit abnormal to have 0 pod in the cluster, but it is not fatal
			if len(podsList) == 0 {
				log.Println("Warning: podList is empty")
			}
			// In practice, the worker pool can't be empty
			if len(nodesList) == 0 {
				log.Println("worker pool should not have 0 nodes, skip this round")
				time.Sleep(schedulerClient.timeInterval * time.Second) //check after the interval
				continue
			}
			//Find unused nodes
			unusedNodes := FindUnusedNodes(podsList,nodesList)
			//Log message
			schedulerClient.DebugMessage(nodesList,podsList,unusedNodes)
			// Make decision to scale in or out the workerPool, using v1 algorithm
			scaleOut,scaleIn,err := SparkAlgoV1(len(nodesList),int(math.Max(0,float64(len(unusedNodes)-FindPendingNodes(podsList)))),schedulerClient.extraNode,
				schedulerClient.minNode,schedulerClient.maxNode)
			if err != nil {
				log.Println(err)
				time.Sleep(schedulerClient.timeInterval * time.Second) //check after the interval
				continue
			}
			if scaleIn{
				removeIndex := rand.Intn(len(unusedNodes))                                   //randomly pick a node to drop
				schedulerClient.ScaleIn(schedulerClient.workerPool,unusedNodes[removeIndex]) //remove the pod
			}else if scaleOut{
				schedulerClient.ScaleOut(schedulerClient.workerPool)
			}
		}else{
			// Auto scaling mode off, turn on maximum number of allowed worker nodes
			log.Println("Cluster AutoScaling is OFF, set the nodes number to max")
			nodesList := schedulerClient.clusterClient.getWorkersNodesIP(schedulerClient.workerPool)
			if len(nodesList) == 0 {
				log.Println("Warning: Node list is empty, skip this round")
				time.Sleep(schedulerClient.timeInterval * time.Second) //check after the interval
				continue
			}
			if len(nodesList) < schedulerClient.maxNode {
				schedulerClient.ScaleOut(schedulerClient.workerPool)
			}
		}
		time.Sleep(schedulerClient.timeInterval * time.Second) //check after the interval
	}
}


/*
Version 1.0 Cluster AutoScaling Algorithm:
1. The unused nodes should always be # of extra nodes until the number of total nodes reach the maximum
2. The number of nodes should >= # of minimum required nodes + # of extra nodes
3. If # of unused nodes > # extra nodes && # of nodes > # of min nodes + # extra nodes, remove 1 node
4. If # of unused nodes < # extra nodes && # of nodes < # max nodes , add 1 node
5. If # of nodes < min nodes + extra, add 1 node
6. If # of nodes > max nodes, best effort to drop the node
 */
func SparkAlgoV1(numOfNodes int, numOfUnusedNodes int, numofExtraNodes int, minNodes int, maxNodes int) (scaleUp bool,
	scaleDown bool, err error) {
	e:= errors.New
	if minNodes > maxNodes {return false,false,e("'MinNode can't be larger than MaxNode")}
	if minNodes + numofExtraNodes > maxNodes{return false,false,e("min + extra can't > max")}
	if numOfNodes < numOfUnusedNodes {return false,false,e("num of unusedNodes can't be larger" +
		"then total nodes")}
	if numOfNodes < (minNodes + numofExtraNodes) {
		return true,false,nil
	}
	if numOfUnusedNodes < numofExtraNodes && numOfNodes < maxNodes{
		return true,false,nil
	}
	if numOfUnusedNodes > numofExtraNodes && numOfNodes > (minNodes + numofExtraNodes) {
		return false,true,nil
	}
	return false,false,nil
}
/*
Randomly remove an unused worker Nodes in a workerPool

Input
-----
workerpoolName:	name of the worker pool
nodeIP: Internal IP address of the node

Output
------
None
 */
func (schedulerClient *Scheduler) ScaleIn(workerpoolName string, nodeIP string) {
	//first try to get the cluster information to exclude network issue
	testInternetConnection := schedulerClient.clusterClient.getClusterResourceGroup()
	if testInternetConnection == ""{
		log.Println("ScaleIn: network problem")
		return
	}
	prevSize := len(schedulerClient.clusterClient.getWorkersNodesIP(schedulerClient.workerPool))
	if prevSize == 0{
		log.Println("Warning: the node list can't empty, skip the action")
		return
	}
	succeed := schedulerClient.clusterClient.removeWorker(workerpoolName,nodeIP)
	if !succeed {
		log.Printf("Node %s in %s can not be removed\n",nodeIP,workerpoolName)
	}else {
		log.Printf("Node %s in %s is being removed\n", nodeIP, workerpoolName)
		timeBegin := time.Now()
		for {
			nodes := schedulerClient.clusterClient.getWorkersNodesIP(schedulerClient.workerPool)
			currSize := len(nodes)	// get the current size
			//TODO: rework on the logic for next version, cuz now any inference on cluster ui might cause an issue
			if currSize == prevSize - 1 {
				canBreak := true
				for _,val := range nodes {
					if val == nodeIP {canBreak = false}
				}
				if canBreak {
					log.Println("Removed worker node takes: ",time.Now().Sub(timeBegin).Minutes()," mins")
					break
				}
			}
			if int(time.Now().Sub(timeBegin).Minutes()) > 10 {break;}
		}
	}
}

/*
Add a worker node in a worker pool

Input
-----
workerpoolName:	name of the worker pool

Output
------
None
*/
func (schedulerClient *Scheduler) ScaleOut(workerpoolName string) {
	// Get the current size of the node list
	testInternetConnection := schedulerClient.clusterClient.getClusterResourceGroup()
	if testInternetConnection == ""{
		log.Println("ScaleOut: network problem")
		return
	}
	prevSize := len(schedulerClient.clusterClient.getWorkersNodesIP(schedulerClient.workerPool))
	if prevSize == 0{
		log.Println("Warning: the node list is empty, skip the action")
		return
	}
	succeed := schedulerClient.clusterClient.addOneWorker(workerpoolName)
	if !succeed {
		log.Println("Can not add a new worker node")
	}else{
		log.Println("Adding a new worker node")
		timeBegin := time.Now()
		for {
			nodes := schedulerClient.clusterClient.getWorkersNodesIP(schedulerClient.workerPool)
			currSize := len(nodes)	// get the current size
			// TODO : also this part, same as scale in
			if currSize == prevSize + 1{
				canBreak := true
				for _,val := range nodes {
					if val == ""{ canBreak = false}
				}
				if canBreak {
					log.Println("Added a new worker node takes: ",time.Now().Sub(timeBegin).Minutes()," mins")
					break
				}
			}
			if int(time.Now().Sub(timeBegin).Minutes()) > 10 {break;}
		}
	}
}

/*
Returns a list of unused worker nodes to be removed in a workerpool,
Note: the unused nodes can be:
1. nodes that are active but idle in the workerPool
2. nodes that are still being provisioned so can't be used by any pod

Input
-----
podlists: list of pods with their attributes
nodelists: list of physical nodes with their internal IP address

Output
------
a list of strings that contains all the unused nodes
 */
func FindUnusedNodes(podlists []apiv1.Pod, nodelists []string)[]string{
	podNodesMap := make(map[string]string)
	var unusedNodes []string
	// build a hash-map to store all the assigned worker nodes and pods' names
	for _, pod := range podlists {
		if pod.Spec.NodeName == ""{
			log.Printf("Warning: the pod %s is assigned to a node with empty IP \n",pod.ObjectMeta.Name)
			continue
		}
		podNodesMap[pod.Spec.NodeName] = pod.ObjectMeta.Name
	}
	for _, nodeIP := range nodelists {
		if _, exist := podNodesMap[nodeIP]; !exist{
			//This worker Node is unused, add to the unused nodes list
			if nodeIP  != ""{
				unusedNodes = append(unusedNodes,nodeIP)
			}
		}
	}
	return unusedNodes
}

/*
Count the number of pods that don't have been assigned to a node
 */
func FindPendingNodes(podlists []apiv1.Pod) int{
	count := 0
	for _, pod := range podlists {
		if pod.Spec.NodeName == ""{
			count++
		}
	}
	return count
}

/*
This function control the period and ON/OFF of auto scaling functions,
in the future if more schedulers are added, we need to write it as a struct for each schedulars

Input
-----
calendar: Monday to Friday auto scaling schedule
timeNow: current time of the call

Output
------
boolean that decides should auto scaling turns on.
 */
func AutoScaleByTime(calendar *[]AutoScalingCalender, timeNow time.Time)bool{
	for _, val := range *calendar{
		// find the matching weekday
		if val.weekday == timeNow.Weekday() {
			// always turns on for the whole up date
			if val.wholeDayOn {return true}
			// two case:
			// 1. start and end time on the same date,
			// 2. start one day and end on the other date

			// same date
			if val.startHour < val.endHour {
				if timeNow.Hour() < val.endHour && timeNow.Hour() >= val.startHour {return true}
				return false
			} else if val.startHour > val.endHour {
				// midnight between start and end hours
				if timeNow.Hour() >= val.startHour || timeNow.Hour() < val.endHour {return true}
				return false
			}
		}
	}
	return false
}

/*
Return the list of pods with matching labels

Input
-----
namespace: by naming convention, should be the name of the workerPool
targetLabels: The key is always ["pool"], the value is the name of the workerPool

Output
------
a list of pods information
 */
func (schedulerClient *Scheduler)GetPodListWithLabels(namespace string, targetLabels map[string]string) []apiv1.Pod {
	podsClient := schedulerClient.clientSet.CoreV1().Pods(namespace)
	set:=labels.Set(targetLabels)
	pods, err := podsClient.List(metav1.ListOptions{
		LabelSelector: set.AsSelector().String(),
	})

	for attempts:=0;err!= nil;{
		if attempts > 2 {
			log.Println(err)
			return nil	// too many tries
		}
		pods, err = podsClient.List(metav1.ListOptions{
			LabelSelector: set.AsSelector().String(),
		})
		attempts = attempts+1
	}
	return pods.Items
}


func( schedulerClient *Scheduler)DebugMessage(nodesList []string,
	podlists []apiv1.Pod,unusedList []string){
	log.Printf("Debug Message: \n\n\n")
	log.Println("nodeslist info")
	for _, value := range nodesList{
			if value == ""{
				log.Println("''")
			}else {log.Println(value)}
	}
	log.Println("podlist info")
	for _, pod := range podlists {
		log.Println(pod.Spec.NodeName," ",pod.ObjectMeta.Name)
	}
	log.Println("unusedList")
	for _, value := range unusedList{
		log.Println(value)
	}
	log.Println("theUnusedNumber is ",len(unusedList)-FindPendingNodes(podlists))
	log.Printf("\n\n")
}