package vault

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/roboll/kube-vault-controller/pkg/kube"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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
		log.Printf("vault-controller: %s: creating secret from path %s", key, claim.Spec.Path)
		err := ctrl.createSecret(key, claim)
		if err != nil {
			log.Printf("vault-controller: %s: created secret from path %s", key, claim.Spec.Path)
		}
		return err
	}

	shouldUpdate := force
	if !shouldUpdate {
		updateTime, err := ctrl.timeUntilUpdate(key, claim, existing)
		if err != nil {
			shouldUpdate = true
			log.Printf("vault-controller: %s: %s (shouldUpdate=%t)", key, err.Error(), shouldUpdate)
		} else if updateTime <= 0 {
			shouldUpdate = true
			log.Printf("vault-controller: %s: %s renew buffer (shouldUpdate=%t)", key, updateTime, shouldUpdate)
		}
	}

	if shouldUpdate {
		renewable, _ := strconv.ParseBool(existing.Annotations[RenewableKey])
		if renewable {
			leaseID := existing.Annotations[LeaseIDKey]
			secret, err := ctrl.tryRenewLease(leaseID)
			if err != nil {
				log.Printf("vault-controller: %s: failed to renew - %s", key, err.Error())
				return ctrl.updateSecret(key, claim)
			}

			log.Printf("vault-controller: %s: lease renewed for %ds", key, secret.LeaseDuration)
			buffer := (time.Duration(claim.Spec.Renew) * time.Second).Seconds()
			if buffer == 0 {
				log.Printf("vault-controller: %s: renew was zero, defaulting to 1h", key)
				buffer = (time.Hour).Seconds()
			}

			if float64(secret.LeaseDuration) > buffer {
				return ctrl.updateSecretMetadata(secret, existing, claim)
			}
			log.Printf("vault-controller: %s: renew duration shorter than renew period, rotating", key)
		}
		return ctrl.updateSecret(key, claim)
	}
	return nil
}

func (ctrl *controller) tryRenewLease(id string) (*vaultapi.Secret, error) {
	if id == "" {
		return nil, errors.New("no lease id")
	}
	return ctrl.vclient.Sys().Renew(id, 0)
}

func (ctrl *controller) updateSecretMetadata(secret *vaultapi.Secret, existing *v1.Secret, claim *kube.SecretClaim) error {
	leaseDuration := time.Duration(secret.LeaseDuration) * time.Second
	leaseExpiration := time.Now().Add(leaseDuration).Unix()
	updated := &v1.Secret{
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
	_, err := ctrl.kclient.Core().Secrets(claim.Namespace).Update(updated)
	return err
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
	return ctrl.kclient.Core().Secrets(claim.Namespace).Delete(claim.Name, &v1.DeleteOptions{})
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

func (ctrl *controller) createSecret(key string, claim *kube.SecretClaim) error {
	secret, err := ctrl.secretForClaim(claim)
	if err != nil {
		return err
	}

	_, err = ctrl.kclient.Core().Secrets(claim.Namespace).Create(secret)
	if err != nil {
		return err
	}
	return nil
}

func (ctrl *controller) updateSecret(key string, claim *kube.SecretClaim) error {
	secret, err := ctrl.secretForClaim(claim)
	if err != nil {
		return err
	}

	_, err = ctrl.kclient.Core().Secrets(claim.Namespace).Update(secret)
	if err != nil {
		return err
	}
	return nil
}

func (ctrl *controller) timeUntilUpdate(key string, claim *kube.SecretClaim, existing *v1.Secret) (time.Duration, error) {
	leaseExpirationString, ok := existing.Annotations[LeaseExpirationKey]
	if !ok {
		return 0, errors.New("needs update, failed to parse lease expiration")
	}

	leaseExpiration, err := strconv.ParseInt(leaseExpirationString, 10, 64)
	if err != nil {
		return 0, errors.New("needs update, failed to parse lease expiration")
	}

	renew := time.Duration(claim.Spec.Renew) * time.Second
	if renew == 0 {
		log.Printf("vault-controller: %s: renew was 0, defaulting to 1h", key)
		renew = time.Hour
	}

	buffer := time.Now().Add(renew)
	expiration := time.Unix(leaseExpiration, 0)
	return expiration.Sub(buffer), nil
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
