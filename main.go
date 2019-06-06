package main

import (
	"flag"
	"log"

	"k8s.io/client-go/tools/clientcmd"

	vault "github.com/hashicorp/vault/api"
	"github.com/roboll/kube-vault-controller/pkg/controller"
	_ "github.com/roboll/kube-vault-controller/pkg/kube/install"
)

var (
	vaddr      = flag.String("vault", "", "(optional) Vault address.")
	apiserver  = flag.String("apiserver", "", "(optional) Kubernetes apiserver url.")
	kubeconfig = flag.String("kubeconfig", "", "Path to the kubeconfig file. Defaults to in-cluster config.")
	namespace  = flag.String("namespace", "", "Namespace to watch for claims.")

	namespacePrefix = flag.String("namespace-prefix", "", "Any claims with this prefix will only be accessible per namespace")

	syncPeriod = flag.Duration("sync-period", 0, "Sync all resources each period.")
)

func main() {
	flag.Parse()

	log.Printf("kube-vault-controller starting, sync period %s.", *syncPeriod)
	if *namespace != "" {
		log.Printf("watching namespace %s.", *namespace)
	}
	if *namespacePrefix != "" {
		log.Printf("all secrets with prefix %s will be namespaced", *namespacePrefix)
	}

	vconfig := vault.DefaultConfig()
	err := vconfig.ReadEnvironment()
	if err != nil {
		panic(err.Error())
	}
	if *vaddr != "" {
		vconfig.Address = *vaddr
	}

	kconfig, err := clientcmd.BuildConfigFromFlags(*apiserver, *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	config := &controller.Config{
		Namespace:  *namespace,
		NamespacePrefix: *namespacePrefix,
		SyncPeriod: *syncPeriod,
	}
	ctrl, err := controller.New(config, vconfig, kconfig)
	if err != nil {
		panic(err.Error())
	}

	stop := make(chan struct{})
	go ctrl.Run(stop)

	<-make(chan struct{})
}
