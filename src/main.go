package main

import (
	"bufio"
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//slice of http|s://hosts/paths build from kubernetes ingress
type url []string

var (
	additional []string
	//excluded built from -customizefile flag.
	excluded []string
	//included regex built from -customizefile flag.
	included []string
	urls     url
)

func main() {
	var (
		config     *rest.Config
		err        error
		customfile *string
		incluster  bool
		kubeconfig *string
	)

	//Start off by getting the config
	//Config defaults to running inside the cluster unless -configfile option is set with full path to kubectl config file
	addurlsfile := flag.String("addurlsfile", "", "path to file with  additional urls to test, 1 line for each url")
	customfile = flag.String("customizefile", "", "path to file with excluded domains, 1 line per domain")
	kubeconfig = flag.String("configfile", "", "absolute path to the kubeconfig file")
	namespace := flag.String("namespace", "", "namespace to monitor ingress objects, defaults all namespaces")
	reloadinterval := flag.Int("reloadinterval", 30, "interval between reloading config from kubernetes in seconds, default 30")
	urlsfile := flag.String("urlfile", "", "path to file with urls to test, 1 line for each url")

	flag.Parse()

	if *kubeconfig == "" {
		incluster = true
	}
	if *customfile != "" && *urlsfile == "" {
		excluded = loadExcluded(customfile)
		included = loadUrlsFromFile(customfile)
	}

	if *addurlsfile != "" {
		additional = loadUrlsFromFile(addurlsfile)
	}

	if incluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *urlsfile == "" {
		//get list of urls from kubernetes
		urls = getIngressUrls(config, *namespace)

		//kick off routine to update hosts slice every -reloadinterval seconds
		go urls.urlUpdater(config, reloadinterval, *namespace)
	} else {
		//load in urls from file
		urls = loadUrlsFromFile(urlsfile)
	}
	//set labels associated with urlStatus metric
	labels := []string{"url"}

	//init metric
	urlStatusMetric := newurlStatusCollector(labels)
	prometheus.MustRegister(urlStatusMetric)

	//start metrics server
	startMetrics()
}

//function for updating list of hosts from kubernetes every i seconds
func (urls *url) urlUpdater(config *rest.Config, i *int, namespace string) {
	for {
		d := time.Duration(*i)
		time.Sleep(d * time.Second)
		*urls = getIngressUrls(config, namespace)
		log.Info("Updated host list.")
	}
}

//build ingress url list from kuberentes
func getIngressUrls(config *rest.Config, namespace string) url {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ingress, err := clientset.Extensions().Ingresses(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	var urls url

	//loop through all ingress items in namespace
	for _, i := range ingress.Items {

		//default http protocol to http
		httpProto := "http"
		//loop through annotations for each ingress object
		for k, v := range i.Annotations {
			//if ssl-redirect is set, only check https
			if k == "ingress.kubernetes.io/ssl-redirect" && v == "true" {
				httpProto = "https"
			} else if k == "traefik.ingress.kubernetes.io/redirect-entry-point" && v == "https" {
				httpProto = "https"
			} else if k == "traefik.ingress.kubernetes.io/frontend-entry-points" && v == "https" {
				httpProto = "https"
			}
		}
		//build list of https urls from ingress.Spec.TLS.hosts[]
		//if ingress.spec.tls.host does not have corresponding ingress.spec.rules.host it is ignored.
		//end result https://ingress.spec.tls.host/ingress.spec.rules.host.path
		for _, t := range i.Spec.TLS {
			//if tls has no host, use rules[0] as the host
			if t.Hosts == nil {
				for _, r := range i.Spec.Rules {
					//skip wild card hosts
					if !strings.Contains(r.Host, "*") {
						for _, p := range r.HTTP.Paths {
							urls = append(urls, "https://"+r.Host+p.Path)
						}
					}
				}
			} else {
				for _, h := range t.Hosts {
					for _, r := range i.Spec.Rules {
						if r.Host == h {
							//skip wild card hosts
							if !strings.Contains(r.Host, "*") {
								for _, p := range r.HTTP.Paths {
									urls = append(urls, "https://"+h+p.Path)
								}
							}
						}
					}
				}
			}
		}

		//if ssl redirect annotation isn't set build http urls from ingress.spce.rules.hosts[].paths[]
		if httpProto != "https" {
			for _, r := range i.Spec.Rules {
				//skip wild card hosts
				if !strings.Contains(r.Host, "*") {
					for _, p := range r.HTTP.Paths {
						urls = append(urls, "http://"+r.Host+p.Path)
					}
				}
			}
		}
	}
	//remove excluded urls in -customizefile file
	urls = urls.removeExcluded(excluded)
	urls = urls.addAdditional(additional)
	return urls

}

//remove excluded urls in -customizefile file
func (urls url) removeExcluded(excluded []string) url {
	for _, exclude := range excluded {
		//loop backwards to avoid skipping elements
		for i := len(urls) - 1; i >= 0; i-- {
			if strings.Contains(urls[i], exclude) {
				//skip if matches included line
				for _, include := range included {
					if !strings.Contains(urls[i], include) {
						urls = append(urls[:i], urls[i+1:]...)
					}
				}
			}
		}
	}
	return urls
}

func (urls url) addAdditional(additional []string) url {
	for _, additional := range additional {
		urls = append(urls, additional)
	}
	return urls
}

//build []string form -customizefile file
func loadExcluded(excludefile *string) []string {
	file, err := os.Open(*excludefile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var excluded []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "!") {
			excluded = append(excluded, strings.Trim(scanner.Text(), "!"))
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return excluded
}

//build []string form -customizefile file
func loadUrlsFromFile(includefile *string) []string {
	file, err := os.Open(*includefile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var u []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "!") {
			u = append(u, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return u
}

//start metrics sever
func startMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	log.Info("Beginning to serve on port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
