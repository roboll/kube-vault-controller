//go:generate codecgen -o types_codec.go types.go
package kube

import (
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	APIGroup        = "vaultproject.io"
	APIVersion      = "v1"
	APIGroupVersion = APIGroup + "/" + APIVersion

	ResourceSecretClaims = "secretclaims"
)

var (
	GroupVersion = unversioned.GroupVersion{
		Group:   APIGroup,
		Version: APIVersion,
	}
)

type SecretSpec struct {
	Type  v1.SecretType          `json:"type"`
	Path  string                 `json:"path"`
	Data  map[string]interface{} `json:"data"`
	Renew int64                  `json:"renew"`
}

type SecretClaim struct {
	unversioned.TypeMeta `json:",inline"`
	api.ObjectMeta       `json:"metadata,omitempty"`

	Spec SecretSpec `json:"spec"`
}

type SecretClaimList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`

	Items []SecretClaim `json:"items"`
}

type SecretClaimManager interface {
	CreateOrUpdateSecret(claim *SecretClaim, force bool) error
	DeleteSecret(claim *SecretClaim) error
}
