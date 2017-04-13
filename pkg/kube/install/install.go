package install

import (
	"fmt"
	"log"

	"github.com/roboll/kube-vault-controller/pkg/kube"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/apimachinery"
	"k8s.io/client-go/pkg/apimachinery/registered"
	"k8s.io/client-go/pkg/apis/certificates/v1alpha1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/sets"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(kube.GroupVersion,
		&kube.SecretClaim{},
		&kube.SecretClaimList{},
		&api.ListOptions{},
		&api.DeleteOptions{},
	)
	return nil
}

const importPrefix = "github.com/roboll/kube-vault-controller/pkg/kube"

var accessor = meta.NewAccessor()

// availableVersions lists all known external versions for this group from most preferred to least preferred
var availableVersions = []unversioned.GroupVersion{kube.GroupVersion}

func init() {
	registered.RegisterVersions(availableVersions)
	externalVersions := []unversioned.GroupVersion{}
	for _, v := range availableVersions {
		if registered.IsAllowedVersion(v) {
			externalVersions = append(externalVersions, v)
		}
	}
	if len(externalVersions) == 0 {
		log.Printf("No version is registered for group %v", kube.APIGroup)
		return
	}

	if err := registered.EnableVersions(externalVersions...); err != nil {
		log.Printf("%v", err)
		return
	}
	if err := enableVersions(externalVersions); err != nil {
		log.Printf("%v", err)
		return
	}
}

func enableVersions(externalVersions []unversioned.GroupVersion) error {
	addVersionsToScheme(externalVersions...)
	preferredExternalVersion := externalVersions[0]

	groupMeta := apimachinery.GroupMeta{
		GroupVersion:  preferredExternalVersion,
		GroupVersions: externalVersions,
		RESTMapper:    newRESTMapper(externalVersions),
		SelfLinker:    runtime.SelfLinker(accessor),
		InterfacesFor: interfacesFor,
	}

	if err := registered.RegisterGroup(groupMeta); err != nil {
		return err
	}
	return nil
}

func newRESTMapper(externalVersions []unversioned.GroupVersion) meta.RESTMapper {
	// the list of kinds that are scoped at the root of the api hierarchy
	// if a kind is not enumerated here, it is assumed to have a namespace scope
	rootScoped := sets.NewString(
		"SecretClaim",
	)

	ignoredKinds := sets.NewString()

	return api.NewDefaultRESTMapper(externalVersions, interfacesFor, importPrefix, ignoredKinds, rootScoped)
}

// interfacesFor returns the default Codec and ResourceVersioner for a given version
// string, or an error if the version is not known.
func interfacesFor(version unversioned.GroupVersion) (*meta.VersionInterfaces, error) {
	switch version {
	case kube.GroupVersion:
		return &meta.VersionInterfaces{
			ObjectConvertor:  api.Scheme,
			MetadataAccessor: accessor,
		}, nil
	default:
		g, _ := registered.Group(kube.APIGroup)
		return nil, fmt.Errorf("unsupported storage version: %s (valid: %v)", version, g.GroupVersions)
	}
}

func addVersionsToScheme(externalVersions ...unversioned.GroupVersion) {
	// add the internal version to Scheme
	if err := AddToScheme(api.Scheme); err != nil {
		// Programmer error, detect immediately
		panic(err)
	}
	// add the enabled external versions to Scheme
	for _, v := range externalVersions {
		if !registered.IsEnabledVersion(v) {
			log.Printf("error: Version %s is not enabled, so it will not be added to the Scheme.", v)
			continue
		}
		switch v {
		case v1alpha1.SchemeGroupVersion:
			if err := v1alpha1.AddToScheme(api.Scheme); err != nil {
				// Programmer error, detect immediately
				panic(err)
			}
		}
	}
}
