package cluster_controller

import (
	"encoding/json"
	"fmt"
	"github.com/json-iterator/go"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

/**
IBM_Cloud_Client struct contains data related to ibm cloud api
 */
type IBMCloudClient struct{
	apiUrl string
	apiKey string
	iamUrl string
	iamToken string
	clusterIdOrName string
	getClusterInfoURI string
	getWorkerPoolsURI string
	getAllWorkersURI string
	resizeOrRebalanceWorkerPoolURI string
	removeWorkerURI string
}

/**
Constructor for IBMCloudClient struct for a cluster
 */
func NewIBMCloudClient() *IBMCloudClient{
	var apiUrl=os.Getenv("IBM_CLOUD_API_URL")
	var iamUrl=os.Getenv("IBM_CLOUD_IAM_URL")
	var apiKey=os.Getenv("IBM_CLOUD_API_KEY")
	var clusterIdOrName=os.Getenv("IBM_CLOUD_CLUSTER_ID_OR_NAME")
	var iamToken=RequestAPIToken(apiKey,iamUrl)
	if iamToken == "" {
		iamToken=os.Getenv("IBM_CLOUD_IAM_TOKEN")
	}
	getClusterInfoURI:="/v1/clusters/"+clusterIdOrName
	getWorkerPoolsURI:="/v1/clusters/"+clusterIdOrName+"/workerpools"
	getAllWorkersURI:="/v1/clusters/"+clusterIdOrName+"/workers"
	resizeOrRebalanceWorkerPoolURI:="/v1/clusters/"+clusterIdOrName+"/workerpools/"
	removeWorkerURI:="/v1/clusters/"+clusterIdOrName+"/workers/"


	client:=&IBMCloudClient{
		apiUrl: apiUrl,
		apiKey: apiKey,
		iamUrl: iamUrl,
		iamToken: iamToken,
		clusterIdOrName: clusterIdOrName,
		getClusterInfoURI: getClusterInfoURI,
		getWorkerPoolsURI: getWorkerPoolsURI,
		getAllWorkersURI: getAllWorkersURI,
		resizeOrRebalanceWorkerPoolURI: resizeOrRebalanceWorkerPoolURI,
		removeWorkerURI: removeWorkerURI,
	}
	return client
}

func (ibmCloudClient *IBMCloudClient) getApiResponse(reqType string, apiUri string, additionalHeader map[string]string, reqBody io.Reader) []byte {
	req, _:= http.NewRequest(reqType, ibmCloudClient.apiUrl+apiUri,reqBody)
	req.Header.Add("Authorization", ibmCloudClient.iamToken)
	req.Header.Add("accept", "application/json")
	for k, v := range additionalHeader{
		req.Header.Add(k,v)
	}
	client := &http.Client{}
	response, err := client.Do(req)

	if err!=nil {
		log.Println(err)
	}
	if response == nil {
		log.Println("Get API response: Got nil response")
		return []byte{}
	}
	if response.Status == "200 OK" {
		// this is for successful GET request
		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		return responseData
	} else if response.Status == "202 Accepted" || response.Status == "204 No Content"{
		// this is for successful POST, PATCH, and delete
		return []byte(response.Status )
	} else if strings.Contains(response.Status, "401") {
		// the token is expired, refresh the token, then recursively finish the http request
		ibmCloudClient.RefreshToken()
		return ibmCloudClient.getApiResponse(reqType,apiUri,additionalHeader,reqBody)
	} else {
		// this is for failed request
		log.Println("GET API response: API request failed")
		return []byte{}
	}
}


/**
This function returns the resource group of the cluster
 */
func (ibmCloudClient *IBMCloudClient) getClusterResourceGroup() string{
	clusterInfo := ibmCloudClient.getApiResponse("GET", ibmCloudClient.getClusterInfoURI, map[string]string{}, nil)
	resourceGroup:=jsoniter.Get(clusterInfo,"resourceGroup",).ToString()
	return resourceGroup
}

/**
This function returns workerpool name list and its from IBM Cloud
 */
func (ibmCloudClient *IBMCloudClient) GetWorkerPools() []string{
	workerPoolsData:=string(ibmCloudClient.getApiResponse("GET", ibmCloudClient.getWorkerPoolsURI, map[string]string{}, nil))
	workerPoolsJson:="{\"workerPool\":"+workerPoolsData+"}"
	jsonIndex:=0
	workerPoolJson:=jsoniter.Get([]byte(workerPoolsJson),"workerPool",jsonIndex).ToString()
	workerPoolNames:= []string{}
	for workerPoolJson!=""{
		workerPoolName:=jsoniter.Get([]byte(workerPoolJson),"name",).ToString()
		workerPoolNames=append(workerPoolNames,workerPoolName)
		jsonIndex+=1
		workerPoolJson=jsoniter.Get([]byte(workerPoolsJson),"workerPool",jsonIndex).ToString()
	}
	return workerPoolNames
}


/**
This function return workers' node IPs
 */
func (ibmCloudClient *IBMCloudClient) getWorkersNodesIP(targetWorkerPoolName string) []string {
	workersInfo:=string(ibmCloudClient.getApiResponse("GET", ibmCloudClient.getAllWorkersURI, map[string]string{}, nil))
	workersInfoJson:="{\"workers\":"+workersInfo+"}"
	jsonIndex:=0
	workerInfoJson:=jsoniter.Get([]byte(workersInfoJson),"workers",jsonIndex).ToString()
	workerNodesIP:=[]string{}
	for workerInfoJson!=""{
		if jsoniter.Get([]byte(workerInfoJson),"poolName",).ToString()==targetWorkerPoolName{
			workerNodeIP:=jsoniter.Get([]byte(workerInfoJson),"privateIP",).ToString()
			workerNodesIP=append(workerNodesIP,workerNodeIP)
		}
		jsonIndex+=1
		workerInfoJson=jsoniter.Get([]byte(workersInfoJson),"workers",jsonIndex).ToString()
	}
	return workerNodesIP
}

/**
This function return workers' node ID
 */
func (ibmCloudClient *IBMCloudClient) getWorkersID(targetWorkerPoolName string, targetWorkerNodeIP string) string {
	workersInfo:=string(ibmCloudClient.getApiResponse("GET", ibmCloudClient.getAllWorkersURI, map[string]string{}, nil))
	workersInfoJson:="{\"workers\":"+workersInfo+"}"
	jsonIndex:=0
	workerInfoJson:=jsoniter.Get([]byte(workersInfoJson),"workers",jsonIndex).ToString()
	targetWorkerID:=""
	for workerInfoJson!=""{
		if jsoniter.Get([]byte(workerInfoJson),"poolName",).ToString()==targetWorkerPoolName{
			if jsoniter.Get([]byte(workerInfoJson),"privateIP",).ToString()==targetWorkerNodeIP{
				targetWorkerID=jsoniter.Get([]byte(workerInfoJson),"id",).ToString()
			}
		}
		jsonIndex+=1
		workerInfoJson=jsoniter.Get([]byte(workersInfoJson),"workers",jsonIndex).ToString()
	}
	return targetWorkerID
}



/**
This function add one worker to the target workerpool
 */
func (ibmCloudClient *IBMCloudClient) addOneWorker(workerPoolName string) bool {
	workerPoolCurrentSize:=len(ibmCloudClient.getWorkersNodesIP(workerPoolName))
	workerPoolTargetSize:=workerPoolCurrentSize+1
	clusterResourceGroup:= ibmCloudClient.getClusterResourceGroup()

	additionalHeader:=make(map[string]string)
	additionalHeader["X-Auth-Resource-Group"]=clusterResourceGroup
	reqData:=fmt.Sprintf(`{
		"sizePerzone":%d,
		"state":"resizing"
	}`, workerPoolTargetSize)
	reqBody := strings.NewReader(reqData)
	reqResult:=string(ibmCloudClient.getApiResponse("PATCH", ibmCloudClient.resizeOrRebalanceWorkerPoolURI+workerPoolName, additionalHeader, reqBody))

	if reqResult=="202 Accepted"{
		return true
	} else {
		return false
	}
}

/*
TODO: this funciton is used to resize the worker pool after add/delete action, however, it does not work as expected due to cloud issue
TODO: So we might reuse it when cloud is stable
This function re-balance the IBM cloud workerPool
*/
func (ibmCloudClient IBMCloudClient) reSize(workerPoolName string) bool {
	workerPoolCurrentSize:=len(ibmCloudClient.getWorkersNodesIP(workerPoolName))
	var workerPoolTargetSize int
	workerPoolTargetSize =workerPoolCurrentSize-1 // need to validate

	clusterResourceGroup:= ibmCloudClient.getClusterResourceGroup()

	additionalHeader:=make(map[string]string)
	additionalHeader["X-Auth-Resource-Group"]=clusterResourceGroup
	reqData:=fmt.Sprintf(`{
		"sizePerzone":%d,
		"state":"resizing"
	}`, workerPoolTargetSize)
	reqBody := strings.NewReader(reqData)
	reqResult:=string(ibmCloudClient.getApiResponse("PATCH", ibmCloudClient.resizeOrRebalanceWorkerPoolURI+workerPoolName, additionalHeader, reqBody))

	if reqResult=="202 Accepted"{
		return true
	} else {
		return false
	}
}

/**
This function add labels to the target workerpool

INPUT:
labelValue: e.g. "pool":"spark"
 */
func (ibmCloudClient *IBMCloudClient) labelWorkerPool(workerPoolName string, labelKey string, labelValue string) bool {
	clusterResourceGroup:= ibmCloudClient.getClusterResourceGroup()

	additionalHeader:=make(map[string]string)
	additionalHeader["X-Auth-Resource-Group"]=clusterResourceGroup
	reqData:=fmt.Sprintf(`{
		"labels": {
			"%s":"%s"
		},
		"state":"labels"
	}`, labelKey, labelValue)
	reqBody := strings.NewReader(reqData)
	reqResult:=string(ibmCloudClient.getApiResponse("PATCH", ibmCloudClient.resizeOrRebalanceWorkerPoolURI+workerPoolName, additionalHeader, reqBody))

	if reqResult=="202 Accepted"{
		return true
	} else {
		return false
	}
}

/**
This function remove one worker from the target workerpool
 */
func (ibmCloudClient *IBMCloudClient) removeWorker(workerpoolName string, nodeIP string) bool {
	targetWorkerID:= ibmCloudClient.getWorkersID(workerpoolName, nodeIP)
	clusterResourceGroup:= ibmCloudClient.getClusterResourceGroup()

	additionalHeader:=make(map[string]string)
	additionalHeader["X-Auth-Resource-Group"]=clusterResourceGroup
	reqResult:=string(ibmCloudClient.getApiResponse("DELETE", ibmCloudClient.removeWorkerURI+targetWorkerID, additionalHeader,nil))

	if reqResult == "204 No Content" {
		return true
	} else{
		return false
	}
}

/*
This function sends a POST request to IBM IAM service and retrieve the API access token given an API Key, the
refresh the value in the ibmCloudClient
*/
func RequestAPIToken(apiKey string, apiUrl string)string{
	// Request for Cloud API Access Token
	data := url.Values{}
	data.Add("grant_type","urn:ibm:params:oauth:grant-type:apikey")
	data.Add("apikey",apiKey)
	req, err := http.NewRequest("POST", apiUrl,strings.NewReader(data.Encode()))
	if err != nil{
		log.Println(err)
	}
	req.Header.Set("Content-Type","application/x-www-form-urlencoded")
	req.Header.Set("Accept","application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	if resp == nil {
		return ""
	}
	defer resp.Body.Close()

	// convert from json string to golang struct
	var postbody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&postbody)
	if err != nil {
		log.Println(err)
	}
	//get acess token
	var accessToken string
	// If does not fetch the token, return an empty string
	if val, ok := postbody["access_token"]; ok {
		accessToken = val.(string)
	}
	return accessToken	//overwrite the token
}

/*
refresh ibm iam token
*/
func (ibmCloudClient *IBMCloudClient)RefreshToken() {
	ibmCloudClient.iamToken=""
	iamToken := RequestAPIToken(ibmCloudClient.apiKey,
		ibmCloudClient.iamUrl)
	if iamToken != "" {
		ibmCloudClient.iamToken = iamToken
		log.Println("Refresh IAM Token: Succeed")
	}else{
		log.Println("Refresh IAM Token: Failed, sleep 1s")
		time.Sleep(1*time.Second)
	}
}