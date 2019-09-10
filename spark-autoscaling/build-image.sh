GOOS=linux go build -o ./service-deployment/spark-custom-autoscaler . && \
ibmcloud cr build --tag registry.ng.bluemix.net/artifactory-autoscaling/spark-autoscaling:sandbox  ./service-deployment
