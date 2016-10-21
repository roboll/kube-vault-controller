package vault

import (
	"fmt"
	"log"
	"strconv"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/roboll/kube-vault-controller/pkg/kube"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/cache"
)

const (
	LeaseIDKey         = "vaultproject.io/lease-id"
	LeaseExpirationKey = "vaultproject.io/lease-expiration"
	RenewableKey       = "vaultproject.io/renewable"

	PKICertificateKey = "certificate"
	PKIPrivateKeyKey  = "private_key"
)

type controller struct {
	vclient *vaultapi.Client
	kclient *kubernetes.Clientset
}

func NewController(vconfig *vaultapi.Config, kconfig *rest.Config) (kube.SecretClaimManager, error) {
	vclient, err := vaultapi.NewClient(vconfig)
	if err != nil {
		return nil, err
	}
	kclient, err := kubernetes.NewForConfig(kconfig)
	if err != nil {
		return nil, err
	}

	return &controller{
		vclient: vclient,
		kclient: kclient,
	}, nil
}

func (ctrl *controller) CreateOrUpdateSecret(claim *kube.SecretClaim, force bool) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(claim)
	if err != nil {
		return err
	}

	existing, err := ctrl.kclient.Core().Secrets(claim.Namespace).Get(claim.Name)
	if err != nil {
		secret, err := ctrl.secretForClaim(claim)
		if err != nil {
			return err
		}
		log.Printf("vault-controller: %s: creating secret from path %s", key, claim.Spec.Path)
		_, err = ctrl.kclient.Core().Secrets(claim.Namespace).Create(secret)
		if err != nil {
			return err
		}
		log.Printf("vault-controller: %s: created secret from path %s", key, claim.Spec.Path)
	} else {
		shouldUpdate := force
		if !shouldUpdate {
			shouldUpdate = ctrl.shouldUpdate(key, claim, existing)
		}
		if shouldUpdate {
			renewable, _ := strconv.ParseBool(existing.Annotations[RenewableKey])
			if renewable {
				leaseID := existing.Annotations[LeaseIDKey]
				if leaseID != "" {
					log.Printf("vault-controller: %s: renewing lease", key)
					secret, err := ctrl.vclient.Sys().Renew(leaseID, 0)
					if err == nil {
						log.Printf("vault-controller: %s: lease renewed for %ds", key, secret.LeaseDuration)
						if float64(secret.LeaseDuration) > (time.Minute).Seconds() { //TODO change time.Hour to spec.Renew
							leaseDuration := time.Duration(secret.LeaseDuration) * time.Second
							leaseExpiration := time.Now().Add(leaseDuration).Unix()
							secret := &v1.Secret{
								ObjectMeta: v1.ObjectMeta{
									Name:      claim.Name,
									Namespace: claim.Namespace,

									Annotations: map[string]string{
										LeaseIDKey:         secret.LeaseID,
										LeaseExpirationKey: strconv.FormatInt(leaseExpiration, 10),
										RenewableKey:       strconv.FormatBool(secret.Renewable),
									},
								},
								Type: existing.Type,
								Data: existing.Data,
							}
							_, err = ctrl.kclient.Core().Secrets(claim.Namespace).Update(secret)
							return err
						} else {
							log.Printf("vault-controller: %s: renew duration shorter than renew period, rotating", key)
						}
					} else {
						log.Printf("vault-controller: %s: unable to renew, rotating", key)
					}
				}
			} else {
				log.Printf("vault-controller: %s: not renewable, rotating", key)
			}
			secret, err := ctrl.secretForClaim(claim)
			if err != nil {
				return err
			}
			_, err = ctrl.kclient.Core().Secrets(claim.Namespace).Update(secret)
			if err != nil {
				return err
			}
			log.Printf("vault-controller: %s: updated secret from path %s", key, claim.Spec.Path)
		} else {
			log.Printf("vault-controller: %s: no update needed for secret", key)
			return nil
		}
	}

	return nil
}

func secretFromVault(claim *kube.SecretClaim, secret *vaultapi.Secret) *v1.Secret {
	leaseDuration := time.Duration(secret.LeaseDuration) * time.Second
	leaseExpiration := time.Now().Add(leaseDuration).Unix()
	return &v1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      claim.Name,
			Namespace: claim.Namespace,

			Annotations: map[string]string{
				LeaseIDKey:         secret.LeaseID,
				LeaseExpirationKey: strconv.FormatInt(leaseExpiration, 10),
				RenewableKey:       strconv.FormatBool(secret.Renewable),
			},
		},
		Type: claim.Spec.Type,
		Data: dataForSecret(claim, secret),
	}
}

func (ctrl *controller) shouldUpdate(key string, claim *kube.SecretClaim, existing *v1.Secret) bool {
	leaseExpirationString, ok := existing.Annotations[LeaseExpirationKey]
	if !ok {
		log.Printf("vault-controller: %s: needs update, failed to parse lease expiration", key)
		return true
	}
	leaseExpiration, err := strconv.ParseInt(leaseExpirationString, 10, 64)
	if err != nil {
		log.Printf("vault-controller: %s: needs update, failed to parse lease expiration", key)
		return true
	}

	buffer := time.Now().Add(time.Hour) //TODO fix time.Hour to spec.Renew
	expiration := time.Unix(leaseExpiration, 0)
	if expiration.Before(buffer) {
		log.Printf("vault-controller: %s: needs update, below minimum lease buffer", key)
		return true
	} else {
		log.Printf("vault-controller: %s: %s left before buffer", key, expiration.Sub(buffer).String())
	}

	return false
}

func (ctrl *controller) secretForClaim(claim *kube.SecretClaim) (*v1.Secret, error) {
	log.Printf("TODO: support authentication per secret")
	logical := ctrl.vclient.Logical()

	var err error
	var value *vaultapi.Secret
	if claim.Spec.Data != nil && len(claim.Spec.Data) > 0 {
		value, err = logical.Write(claim.Spec.Path, claim.Spec.Data)
	} else {
		value, err = logical.Read(claim.Spec.Path)
	}

	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("no secret found for %s", claim.Spec.Path)
	}

	return secretFromVault(claim, value), nil
}

func dataForSecret(claim *kube.SecretClaim, secret *vaultapi.Secret) map[string][]byte {
	data := make(map[string][]byte, len(secret.Data))
	switch claim.Spec.Type {
	case v1.SecretTypeTLS:
		data[v1.TLSCertKey] = []byte(secret.Data[PKICertificateKey].(string))
		data[v1.TLSPrivateKeyKey] = []byte(secret.Data[PKIPrivateKeyKey].(string))
	default:
		for key, val := range secret.Data {
			datom, _ := val.(string)
			data[key] = []byte(datom)
		}
	}
	return data
}

func (ctrl *controller) DeleteSecret(claim *kube.SecretClaim) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(claim)
	if err != nil {
		return err
	}

	log.Printf("vault-controller: revoking lease for secret %s", key)
	secret, err := ctrl.kclient.Core().Secrets(claim.Namespace).Get(claim.Name)
	if err != nil {
		log.Printf("vault-controller: %s: not revoking, failed to get secret for deleted claim: %s", key, err.Error())
	} else {
		leaseID, ok := secret.Annotations[LeaseIDKey]
		if !ok {
			log.Printf("vault-controller: %s: not revoking, failed to get lease id to revoke", key)
		} else if leaseID == "" {
			log.Printf("vault-controller: %s: not revoking, no lease id annotation", key)
		} else {
			if err := ctrl.vclient.Sys().Revoke(leaseID); err != nil {
				log.Printf("vault-controller: %s: failed to revoke lease id %s: %s", key, leaseID, err.Error())
			} else {
				log.Printf("vault-controller: %s: revoked lease id %s", key, leaseID)
			}
		}
	}

	log.Printf("vault-controller: %s: deleting secret", key)
	return ctrl.kclient.Core().Secrets(claim.Namespace).Delete(claim.Name, &api.DeleteOptions{})
}
