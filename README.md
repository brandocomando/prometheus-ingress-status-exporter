# Ingress Status Exporter for Prometheus

This is a simple server that scrapes ingress objects stored in kubernetes to generate and test a list of urls and exports the http response code for each of them via HTTP for Prometheus consumption.

What exactly is it doing?
The Biggest assumption that I am making with this project is that if you create an ingress object with hosts and paths they should be reachable with a http/s request, and you would want to monitor them with prometheus.
This project is most likely useful for those of us using an ingress controller such as [haproxy](https://github.com/jcmoraisjr/haproxy-ingress), [nginx](https://github.com/kubernetes/ingress-nginx), or [traefik](https://docs.traefik.io/user-guide/kubernetes/) in kubernetes. But with some flags explained below it can be configured as a simple url status checker and any list of urls.
My scope of how others are using ingress controllers is limited so if this project does not work for your implementation of an ingress controller/ingress objects please let me know.
see [Ingress Parsing](#Ingress-Parsing) section below to see the details of how urls are generated.

## Getting Started

To run it:

```bash
./ingress-status-exporter [flags]
```

Help on flags:

```bash
./ingress-status-exporter --help
```

## Usage

### Configfile
by default ingress-status-exporter assumes its running inside a kubernetes cluster and will require read permissions to ingress objects on the namespace 
But if you would like to run outside of the cluster kubectl needs to be installed and configured, you can pass in the config file like so:
```bash
./ingress-status-exporter -configfile FullPath/To/Kubectl/ConfigFile
```
### Reload Interval

This flag allows you to set the interval at which the ingress objects are queried
passed as number of seconds. default is 30.
```bash
./ingress-status-exporter -reloadinterval 30
```
### Namespace
You can specify a specific namespace to check with the -namespace flag. Defaults to all namespaces
```bash
./ingress-status-exporter -namespace default
```

### Urlsfile
by default ingress-status-exporter will generate a list of urls from kubernetes ingress objects, but if you would like to specify a list of urls to check you can do so with the -urlfile flag
```bash
./ingress-status-exporter -urlfile FullPath/To/File
```
the url file should contain a single url per line:
```
https://google.com
https://golang.org/pkg/
```
### Addurlsfile
While -urlsfile will pervent loading urls from kubernetes, -addurlsfile will simple add additional urls to check. The format ix exactly the same as urlsfile.
```
https://google.com
https://golang.org/pkg/
```
will check all the urls generated from kubernetes and https://google.com and https://golang.org/pkg/

### Customize
Admittedly the most confusing part of this application is the -customizefile flag.
This takes a text file similar to the urlfile flag with should look like so:
```
!google.com
!golang.org
golang.org/pkg
```
lines starting with ! are exclusions and lines not starting with ! are exceptions to those exclusions.
So the example above would prevent \*google.com\* from being checked and \*golang.org\* but would allow golang.org/pkg.
This can be handy if you have some ingress urls that maybe accept traffic on certain paths for health checking, but not other paths you have set in your ingress definition. 

## Ingress Parsing
This exporter will check a list built from ingress kubernetes objects. It will build a http url for every host/path under spec.rules, and an https url for every host under spec.tls for every path under rules/host/path for the corrisponding host. That is probably confusing so heres some examples:
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example
spec:
  tls:
  - hosts:
    - test.example.com
    secretName: tls-secret
  rules:
  - host: test.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-1
          servicePort: 8080
      - path: /ping
        backend:
          serviceName: example-svc-1
          servicePort: 8080
```
Parsing this file will build the following urls to check
```
http://test.example.com
https://test.example.com
http://test.example.com/ping
https://test.example.com/ping
```

if we wanted to add some more hosts/rules we can do that as well
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example
spec:
  tls:
  - hosts:
    - test-1.example.com
    secretName: tls-secret
    - test-2.example.com
    secretName: tls-secret
  rules:
  - host: test-1.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-1
          servicePort: 8080
      - path: /ping1
        backend:
          serviceName: example-svc-1
          servicePort: 8080
  - host: test-2.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-2
          servicePort: 8080
      - path: /ping2
        backend:
          serviceName: example-svc-2
          servicePort: 8080
```
This will create the following urls to check:
```
http://test-1.example.com
https://test-1.example.com
http://test-1.example.com/ping1
https://test-1.example.com/ping1
http://test-2.example.com
https://test-2.example.com
http://test-2.example.com/ping2
https://test-2.example.com/ping2
```
any of the following annotations will remove the http check and only check https:
```
ingress.kubernetes.io/ssl-redirect
traefik.ingress.kubernetes.io/redirect-entry-point
traefik.ingress.kubernetes.io/frontend-entry-points
```
for example:
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example
  annotations:
    ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - test-1.example.com
    secretName: tls-secret
    - test-2.example.com
    secretName: tls-secret
  rules:
  - host: test-1.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-1
          servicePort: 8080
      - path: /ping1
        backend:
          serviceName: example-svc-1
          servicePort: 8080
  - host: test-2.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-2
          servicePort: 8080
      - path: /ping2
        backend:
          serviceName: example-svc-2
          servicePort: 8080
```
checks:
```
https://test-1.example.com
https://test-1.example.com/ping1
https://test-2.example.com
https://test-2.example.com/ping2
```

WildCard hosts in traefik ingress objects are ignored, for example:
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: wildcard-cheeses
  annotations:
    traefik.frontend.priority: "1"
spec:
  rules:
  - host: *.minikube.com
    http:
      paths:
      - path: /
        backend:
          serviceName: stilton
          servicePort: http

kind: Ingress
metadata:
  name: specific-cheeses
  annotations:
    traefik.frontend.priority: "2"
spec:
  rules:
  - host: specific.minikube.com
    http:
      paths:
      - path: /
        backend:
          serviceName: stilton
          servicePort: http
```
would only check:
```
http://specific.minikube/
https://specific.minikube/
```

If a host is not specified under tls, it builds an https check for host/path in rules:
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example
  annotations:
    ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - secretName: tls-secret
  rules:
  - host: test-1.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-1
          servicePort: 8080
      - path: /ping1
        backend:
          serviceName: example-svc-1
          servicePort: 8080
  - host: test-2.example.com
    http:
      paths:
      - path: 
        backend:
          serviceName: example-svc-2
          servicePort: 8080
      - path: /ping2
        backend:
          serviceName: example-svc-2
          servicePort: 8080

```
```
https://test-1.example.com
https://test-1.example.com/ping1
https://test-2.example.com
https://test-2.example.com/ping2
```
## Development
### Docker
you can find the built images here: [dockerhub](https://hub.docker.com/r/brandocomando8/ingress-status-exporter/)
### Building

```bash
./build_locally.sh
```
or with docker
```bash
docker build ./ -t ingress-status-exporter
```

## Pull requests
Please feel free to submit pull requests. This is my first real venture into go and really coding in general, so I know I have a lot to learn and more than open to learning how this project can be improved.
Also if you have feature requests, let me know as well, im looking for more challenges.
## License

Apache 2.0, see [LICENSE](LICENSE)
