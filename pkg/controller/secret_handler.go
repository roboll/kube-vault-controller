package controller

import (
	"log"
	"reflect"

	"github.com/roboll/kube-vault-controller/pkg/kube"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func newSecretHandler(manager kube.SecretClaimManager, claims cache.Store) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, obj interface{}) {
			handleSecretOp(manager, claims, obj, "update")
		},
		DeleteFunc: func(obj interface{}) {
			handleSecretOp(manager, claims, obj, "delete")
		},
	}
}

func handleSecretOp(manager kube.SecretClaimManager, claims cache.Store, obj interface{}, op string) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}

	log.Printf("secret-handler: %s: handling %s for secret", key, op)
	raw, exists, err := claims.GetByKey(key)
	if err != nil {
		log.Printf("error: failed to get claim by key (%s): %s.", key, err.Error())
		return
	}
	if !exists {
		log.Printf("secret-handler: %s: skipping secret %s, no claim found", key, op)
		return
	}

	claim, ok := raw.(*kube.SecretClaim)
	if !ok {
		log.Printf("error: expected *kube.SecretClaim, got %s", reflect.TypeOf(obj))
		return
	}
	log.Printf("secret-handler: %s: requesting secret create/update (force=false)", key)
	if err := manager.CreateOrUpdateSecret(claim, false); err != nil {
		log.Printf("error: failed to create/update secret %s: %s", key, err.Error())
	}
}

// newSecretSoruce returns a cache.ListerWatcher for secret objects.
func newSecretSource(config *rest.Config, namespace string) (cache.ListerWatcher, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	secretClient := clientset.Core().RESTClient()
	return cache.NewListWatchFromClient(secretClient, "secrets", namespace, nil), nil
}
