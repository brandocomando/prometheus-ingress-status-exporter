### A Simple Example of running inside a kubernetes cluster

This example assumes you already have a working [kubernetes](https://github.com/kubernetes/kubernetes) cluster and [prometheus](https://github.com/coreos/prometheus-operator) up and running as well.
I'm running my exporter in the monitoring namespace, but you can change it to run where ever your prometheus setup can find it. 

I've provided a simple configmap for the customization [file](ingress-status-exporter-cm.yaml), you can expand on this configmap to include other files you wish to use as command line arguments.

so to get started apply the rbac settings allowing the default service account for the monitoring namespace to read ingress objects.
```
kubeclt apply -f kubernetes/ingress-reader-rbac.yaml
```

next we'll create the configmap:
```
kubectl apply -f kubernetes/ingress-status-exporter-cm.yaml
```

then the deployment and service:
```
kuebctl apply -f kubernetes/ingress-status-exporter-deploy.yaml
kuebctl apply -f kubernetes/ingress-status-exporter-svc.yaml
```

and lastly we can create the service monitor to tell prometheus to gather our stats.
```
kubectl apply -f kubernetes/ingress-status-exporter-sm.yaml
```
This should be enough to get you up and running, you should start to see your urlStatus metrics showing up in prometheus, and from there you can see what adjustments you'd like to make via the flag options. 

### Running Locally Via Docker
This example runs the exporter locally using a provided list of urls.
you can use the provided urlsfile in docker/volume list so:
```
docker run -v ${PWD}/docker/volume:/opt/config/ -it -p 8080:8080  brandocomando8/ingress-status-exporter -urlfile /opt/config/url_test.txt
curl localhost:8080/metrics
```
you should see some output like:
```
# HELP urlStatus Shows if a site is up
# TYPE urlStatus counter
urlStatus{url="https://golang.org"} 200
urlStatus{url="https://google.com"} 200
urlStatus{url="https://http://brokenexample.com/"} 523
```

### Running Locally
I've provided a build_locally.sh script in the root directory of the repo. Provided you have go installed and configured it will build the binary for you. 
once you have the binary you can run it locally via the command line:
```
./ingress-status-exporter -h
```

