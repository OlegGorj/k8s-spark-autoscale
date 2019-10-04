## How to unit test in out of cluster environment 
To do unit testing outside of kubernetes cluster, the following 
steps are needed:
1. Open a terminal, and run `ibmcloud login -sso`
2. A web page will be opened automatically, copy paste the one time
authentication password in your terminal login section. Once validation 
get passed, select your IBM cloud account and region.
3. Run `ibmcloud ks cluster-config --cluster <cluster name>` to get the config 
file for that cluster.
4. Add the environment variable by pasting the path under `Export environment variables to start using Kubernetes`
into ~/.profile or ~/.bashrc in the format:
`export KUBECONFIG_ABSOLUTE_PATH=(path)`
5. run `source ~/.profile` or `source ~/.bashrc`
6. Now, you can create a out of cluster k8s client in the test
 
The point is that the local application needs to be aware of the location of 
kubernetes config, so alternatively (or if previous steps don't work), do:
1. Repeat step 1-3 if the config is not existed locally or expired
2. Open Goland IDE, in Edit Configuration, select the build you are going 
to use, and in `Environment` option, add `KUBECONFIG_ABSOLUTE_PATH=<path>` where the path is given on the terminal after running the commends in step1
3. Similarly, the other environment variables can be added under the `Environment` option

## How to run the service locally
1. Make sure the kubernetes config file is specified in environment variables
2. Forward the services in Cluster to your local port
3. Run the app
