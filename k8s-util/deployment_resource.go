package k8s_util

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apiv1 "k8s.io/api/core/v1"
)


type DeploymentResource struct {
	Cores        string
	Mem          string
	ContainerCpu string
	ContainerMem string
}

func NewDeploymentResource(cores string,mem string,containerCpu string,containerMem string) *DeploymentResource{

	return &DeploymentResource{
		Cores:        cores,
		Mem:          mem,
		ContainerCpu: containerCpu,
		ContainerMem: containerMem,
	}
}

func (deploymentResource DeploymentResource) GenerateResourceRequirements() apiv1.ResourceRequirements{
	return apiv1.ResourceRequirements{
		Limits: map[apiv1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse(deploymentResource.ContainerCpu),
			"memory": resource.MustParse(deploymentResource.ContainerMem),
		},
		Requests: map[apiv1.ResourceName]resource.Quantity{
			"cpu":    resource.MustParse(deploymentResource.ContainerCpu),
			"memory": resource.MustParse(deploymentResource.ContainerMem),
		},
	}
}


