package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	k8s_acme_cache "github.com/lstoll/k8s-acme-cache"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getBoolEnv(varname string) bool {
	result := os.Getenv(varname)
	switch result {
	case "false", "FALSE", "False", "0":
		return false
	default:
		return true
	}
}

var domain = flag.String("domain", "", "The domain to use")
var email = flag.String("email", "", "The email registering the cert")
var port = flag.Int("port", 8443, "The port to listen on")

var staging = flag.Bool("staging", getBoolEnv("STAGING"), "Use the letsencrypt staging server")

var incluster = flag.Bool("incluster", getBoolEnv("IN_CLUSTER"), "Specify if the application is running in a cluster")
var kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")

var namespace = flag.String("namespace", "", "Namespace to use for cert storage.")
var secretName = flag.String("secret", "acme.secret", "Secret to use for cert storage")

func createInClusterClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func createExternalClient(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func getNamespace() string {
	if len(*namespace) > 0 {
		return *namespace
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return "default"
}

func main() {
	flag.Parse()
	var client *kubernetes.Clientset
	var err error
	if *incluster {
		client, err = createInClusterClient()
	} else {
		client, err = createExternalClient(*kubeconfig)
	}
	if err != nil {
		panic(err)
	}

	cache := k8s_acme_cache.New(
		getNamespace(),
		*secretName,
		client,
		1,
	)
	var acmeClient *acme.Client
	if *staging {
		acmeClient = &acme.Client{DirectoryURL: "https://acme-staging.api.letsencrypt.org/directory"}
	}

	log.Printf("Creating cert manager for domain %s", *domain)
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*domain), //your domain here
		Cache:      cache,
		Email:      *email,
		Client:     acmeClient,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world"))
		log.Printf("Got request to %s", r.URL.String())
	})

	portString := fmt.Sprintf(":%d", *port)

	server := &http.Server{
		Addr: portString,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}
	log.Printf("listening on %s", server.Addr)
	log.Fatal(server.ListenAndServeTLS("", "")) //key and cert are comming from Let's Encrypt

}
