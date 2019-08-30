# Custom Autoscaler

## Spark Custom Autoscaler
### Behavior description:
This spark custom autoscaler is primarily used for jhub. On the one hand, the spark cluster will keep autoscaling to make sure there are always some idle spark workers for jhub users. Let the number of current spark workers in use be A, the number of idle spark workers set by us be I, then this spark autoscaler will keep
the total number of the spark workers in the cluster being (A+I) as long as there is enough computing resource allocated for the spark autoscaler deployment.
On the other hand, this spark autoscaler guarantees the spark workers used by jhub's users will never be killed while the spark cluster is scaling in. In other words, the spark drivers in jhub should never experience "worker loss" when the spark cluster scales in.

### Usage
Change the current directory to `custom-autoscaling/spark/service-deployment` and run
```$xslt
kubectl apply -f spark-custom-autoscaler.yaml -n spark
```

### How spark custom autoscaler works
This custom autoscaler is implemented with k8s go-client. There are two primary components in this autoscaler, including `SparkMasterDeployment` and `SparkMasterDeployment`. Both these two components communicate with k8s cluster through `DeploymentClient`.

`SparkMasterDeployment` create the necessary services (Spark UI and Spark service) and then create the k8s Deployment of Spark master, where the Spark UI service binds the Spark master UI (port 8080) and Spark service binds the Spark master service (with port 7077).

 `SparkWorkerDeployment` uses https request to retrieve information in json format about the Spark cluster through the Spark UI service, to decide whether to add more Spark workers to the cluster or kill workers in the cluster. While `SparkWorkerDeployment` is trying to add a worker to the cluster, it just generates the worker's configuration and send the configuration with pod creation request throught `DeploymentClient`, then the worker/pod will try to join to the Spark cluster through Spark master service. When `SparkWorkerDeployment` needs to kill a worker in the cluster, it will check the utilization of workers and pick a worker without any CPU usage, then send the pod deleting request throught `DeploymentClient`.

 One important problem while we are implementing this autoscaler is to find out the relationship between the worker id and pod name of a Spark worker, because we haven't found a way to set the Spark worker id. However, we find that the worker id contains the information about the node ip and port which a Spark worker uses to communicate with the Spark master, so the solution for this problem here is to check the log of each worker pod to find out the k8s node ip and port, then build up a map between worker id and pod name.

 ### Introduction to further development for Spark custom autoscaler
 The autoscaler is implemented in golang and dependent on k8s go-client heavily. To further develop this autoscaler, you need to set up golang in your  local side firstly. The main function is named `spark_deployment_service.go`. Notice that you need to set the environment variable `IS_IN_CLUSTER=false` and `KUBECONFIG_ABSOLUTE_PATH` such that the k8s go-client can retrieve your kubeconfig from your local side. The testing function for this autoscaler is in `spark/autoscaling_test.go` which is used to test if the autoscaler behaves as the expected way.

 After finishing the further development, you can run
 ```$xslt
GOOS=linux go build -o ./service-deployment/spark-custom-autoscaler .
```
to build the app. Then change your current directory to `spark/service-deployment` to build and push a new image with docker. Finally, use
```$xslt
kubectl apply -f spark-custom-autoscaler.yaml -n spark
```
to deploy your new Spark autoscaler.
