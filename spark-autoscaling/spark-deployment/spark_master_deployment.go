package spark_deployment

import (
	"github.ibm.com/AdvancedAnalyticsCanada/custom-autoscaling/k8s-util"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"os"
)
/**
This struct contains data related to spark master
 */
type SparkMasterDeployment struct{
	deploymentClient *k8s_util.DeploymentClient
	image_name string
	labels map[string]string
	nodeSelector map[string]string
	sparkMasterName string
	sparkPath string
	sparkMasterSerivcePort string
	sparkMasterWebuiPort string
	deploymentConfig *appsv1.Deployment
	deploymentResource *k8s_util.DeploymentResource
}

/**
Constructor for SparkMasterDeployment
 */
func NewSparkMasterDeployment(deploymentClient *k8s_util.DeploymentClient,
	image_name string,
	resource *k8s_util.DeploymentResource) *SparkMasterDeployment{
	return &SparkMasterDeployment{
		deploymentClient: deploymentClient,
		image_name:image_name,
		sparkPath: "/usr/spark",
		labels:map[string]string{
			"component": "spark-master",
			"pool": os.Getenv("SPARK_MASTER_POOL"),
		},
		nodeSelector:map[string]string{
			"pool": os.Getenv("SPARK_MASTER_POOL"),
		},
		sparkMasterName: "spark-master",
		sparkMasterSerivcePort:"7077",
		sparkMasterWebuiPort: "8080",
		deploymentResource: resource,
	}
}

/**
This function is used to clean the existing Spark master deployment and deploy a new one
 */
func (sparkMasterDeployment *SparkMasterDeployment) Deploy(){
	sparkMasterDeployment.deploymentClient.DeleteService("spark-master")
	sparkMasterDeployment.deploymentClient.DeleteService("spark-webui")
	sparkMasterDeployment.deploymentClient.DeleteDeployment(sparkMasterDeployment.sparkMasterName)

	sparkMasterDeployment.deploymentClient.CreateService("spark-master", 7077, sparkMasterDeployment.labels)
	sparkMasterDeployment.deploymentClient.CreateService("spark-webui", 8080, sparkMasterDeployment.labels)
	sparkMasterDeployment.deploymentClient.CreateDeployment(
		sparkMasterDeployment.generateDeploymentConfig())
}


/**
This function is to generate k8s deployment configuration for Spark master
 */
func (sparkMasterDeployment *SparkMasterDeployment) generateDeploymentConfig()(* appsv1.Deployment){
	spark_master_deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: sparkMasterDeployment.sparkMasterName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: k8s_util.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: sparkMasterDeployment.labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: sparkMasterDeployment.labels,
				},
				Spec: apiv1.PodSpec{
					ImagePullSecrets: []apiv1.LocalObjectReference{
						{Name: "image-pull-secret-ibm-cloud"},
					},
					Containers: []apiv1.Container{
						{
							Name:  "spark-master",
							Image: sparkMasterDeployment.image_name,
							Command: []string{
								"/bin/sh",
								"-c",
							},
							Args: []string{
								//"echo $(hostname -i) "+sparkMasterDeployment.sparkMasterName+" >> /etc/hosts && python3 -m http.server",
								"echo $(hostname -i) "+sparkMasterDeployment.sparkMasterName+" >> /etc/hosts && "+sparkMasterDeployment.sparkPath+"/bin/spark-class org.apache.spark.deploy.master.Master",
							},
							Ports: []apiv1.ContainerPort{
								{
									Name:          "master",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 7077,
								},
								{
									Name:          "webui",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
							Env: []apiv1.EnvVar{
								{
									Name: "SPARK_DAEMON_MEMORY",
									Value: "1g",
								},
								{
									Name:  "SPARK_MASTER_HOST",
									Value: sparkMasterDeployment.sparkMasterName,
								},
								{
									Name:  "SPARK_MASTER_PORT",
									Value: sparkMasterDeployment.sparkMasterSerivcePort,
								},
								{
									Name:  "SPARK_MASTER_WEBUI_PORT",
									Value: sparkMasterDeployment.sparkMasterWebuiPort,
								},
							},
							Resources: sparkMasterDeployment.deploymentResource.GenerateResourceRequirements(),
						},
					},
					NodeSelector: sparkMasterDeployment.nodeSelector,
				},
			},
		},
	}
	return spark_master_deployment
}