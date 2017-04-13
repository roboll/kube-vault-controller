package controller

import (
	"log"
	"reflect"

	"github.com/roboll/kube-vault-controller/pkg/kube"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func newSecretClaimHandler(manager kube.SecretClaimManager) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Printf("error: failed to get key for obj, dropping.")
				return
			}

			log.Printf("secret-claim-handler: %s: handling add for secret claim", key)
			claim, ok := obj.(*kube.SecretClaim)
			if !ok {
				log.Printf("error: expected *kube.SecretClaim, got %s", reflect.TypeOf(obj))
				return
			}

			log.Printf("secret-claim-handler: %s: scheduling create/update for secret (force=true)", key)
			if err := manager.CreateOrUpdateSecret(claim, true); err != nil {
				log.Printf("error: failed to update secret for key %s: %s", key, err.Error())
			}
		},
		UpdateFunc: func(old, obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Printf("error: failed to get key for obj, dropping.")
				return
			}

			log.Printf("secret-claim-handler: %s: handling update for secret claim", key)
			claim, ok := obj.(*kube.SecretClaim)
			if !ok {
				log.Printf("expected *kube.SecretClaim, got %s", reflect.TypeOf(obj))
				return
			}
			oldClaim, ok := old.(*kube.SecretClaim)
			if !ok {
				log.Printf("expected *kube.SecretClaim, got %s", reflect.TypeOf(old))
				return
			}

			force := !reflect.DeepEqual(claim.Spec, oldClaim.Spec)
			log.Printf("secret-claim-handler: %s: scheduling create/update for secret (force=%t)", key, force)
			if err := manager.CreateOrUpdateSecret(claim, force); err != nil {
				log.Printf("error: failed to update secret for key %s: %s", key, err.Error())
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				log.Printf("error: failed to get key for obj, dropping.")
				return
			}

			log.Printf("secret-claim-handler: %s: handling delete for secret claim", key)
			claim, ok := obj.(*kube.SecretClaim)
			if !ok {
				log.Printf("error: expected *kube.SecretClaim, got %s", reflect.TypeOf(obj))
				return
			}

			log.Printf("secret-claim-handler: %s: scheduling delete for secret", key)
			if err := manager.DeleteSecret(claim); err != nil {
				log.Printf("error: failed to delete secret for key %s: %s", key, err.Error())
			}
		},
	}
}

// newClaimSource returns a cache.ListerWatcher for secret claim objects.
func newSecretClaimSource(config *rest.Config, namespace string) (cache.ListerWatcher, error) {
	configCopy := *config
	if configCopy.UserAgent == "" {
		configCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	configCopy.APIPath = "/apis"
	configCopy.GroupVersion = &kube.GroupVersion
	configCopy.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	client, err := rest.RESTClientFor(&configCopy)
	if err != nil {
		return nil, err
	}

	return cache.NewListWatchFromClient(client, kube.ResourceSecretClaims, namespace, nil), nil
}
