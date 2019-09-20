GOOS=linux go build -o ./cluster-deployment/cluster-autoscaler . && \
ibmcloud cr build --tag registry.ng.bluemix.net/artifactory-autoscaling/cluster-autoscaling:dev  ./cluster-deployment
