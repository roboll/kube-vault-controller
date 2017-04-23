package controller

import (
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/roboll/kube-vault-controller/pkg/kube"
	"github.com/roboll/kube-vault-controller/pkg/vault"

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	SecretController      *cache.Controller
	SecretClaimController *cache.Controller
}

type Config struct {
	Namespace  string
	SyncPeriod time.Duration
}

func New(config *Config, vconfig *vaultapi.Config, kconfig *rest.Config) (*Controller, error) {
	claimSource, err := newSecretClaimSource(kconfig, config.Namespace)
	if err != nil {
		return nil, err
	}
	secretSource, err := newSecretSource(kconfig, config.Namespace)
	if err != nil {
		return nil, err
	}
	vaultController, err := vault.NewController(vconfig, kconfig)
	if err != nil {
		return nil, err
	}

	claims, claimCtrl := cache.NewInformer(claimSource, &kube.SecretClaim{}, config.SyncPeriod, newSecretClaimHandler(vaultController))
	_, secretCtrl := cache.NewInformer(secretSource, &v1.Secret{}, 0, newSecretHandler(vaultController, claims))

	return &Controller{
		SecretController:      secretCtrl,
		SecretClaimController: claimCtrl,
	}, nil
}

func (ctrl *Controller) Run(stop chan struct{}) {
	secretStop := make(chan struct{})
	go ctrl.SecretController.Run(secretStop)

	claimStop := make(chan struct{})
	go ctrl.SecretClaimController.Run(claimStop)

	<-stop
	secretStop <- struct{}{}
	claimStop <- struct{}{}
}
