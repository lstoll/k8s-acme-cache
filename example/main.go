package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/micahhausler/k8s-acme-cache"
	flag "github.com/spf13/pflag"
	"golang.org/x/crypto/acme/autocert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var domain = flag.StringArray("domain", []string{}, "The domain to use")
var email = flag.String("email", "", "The email registering the cert")
var port = flag.Int("port", 8443, "The port to listen on")

var kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")

var namespace = flag.String("namespace", "", "Namespace to use for cert storage.")
var secretName = flag.String("secret", "acme.secret", "Secret to use for cert storage")

func createClient(kubeconfig string) *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
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
	client := createClient(*kubeconfig)

	cache := k8s_acme_cache.KubernetesCache(
		*secretName,
		getNamespace(),
		client,
	)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*domain...), //your domain here
		Cache:      cache,                              //folder for storing certificates
		Email:      *email,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello world"))
	})

	//cert, err := certManager.GetCertificate
	portString := fmt.Sprintf(":%d", *port)

	server := &http.Server{
		Addr: portString,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}
	log.Fatal(server.ListenAndServeTLS("", "")) //key and cert are comming from Let's Encrypt

}
