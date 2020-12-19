package oc

import (
	"github.com/openshift/api/apps"
	"github.com/openshift/api/authorization"
	"github.com/openshift/api/build"
	"github.com/openshift/api/image"
	"github.com/openshift/api/network"
	"github.com/openshift/api/oauth"
	"github.com/openshift/api/project"
	quotav1 "github.com/openshift/api/quota/v1"
	"github.com/openshift/api/route"
	securityv1 "github.com/openshift/api/security/v1"
	"github.com/openshift/api/template"
	"github.com/openshift/api/user"
	"github.com/openshift/oc/pkg/helpers/legacy"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"os"
	"path/filepath"
)

func InitiatedCommand() *cobra.Command {
	// the kubectl scheme expects to have all the recognizable external types it needs to consume.  Install those here.
	// We can't use the "normal" scheme because apply will use that to build stategic merge patches on CustomResources
	utilruntime.Must(apps.Install(scheme.Scheme))
	utilruntime.Must(authorization.Install(scheme.Scheme))
	utilruntime.Must(build.Install(scheme.Scheme))
	utilruntime.Must(image.Install(scheme.Scheme))
	utilruntime.Must(network.Install(scheme.Scheme))
	utilruntime.Must(oauth.Install(scheme.Scheme))
	utilruntime.Must(project.Install(scheme.Scheme))
	utilruntime.Must(installNonCRDQuota(scheme.Scheme))
	utilruntime.Must(route.Install(scheme.Scheme))
	utilruntime.Must(installNonCRDSecurity(scheme.Scheme))
	utilruntime.Must(template.Install(scheme.Scheme))
	utilruntime.Must(user.Install(scheme.Scheme))
	legacy.InstallExternalLegacyAll(scheme.Scheme)

	basename := filepath.Base(os.Args[0])
	return CommandFor(basename)
}

func installNonCRDSecurity(scheme *apimachineryruntime.Scheme) error {
	scheme.AddKnownTypes(securityv1.GroupVersion,
		&securityv1.PodSecurityPolicySubjectReview{},
		&securityv1.PodSecurityPolicySelfSubjectReview{},
		&securityv1.PodSecurityPolicyReview{},
		&securityv1.RangeAllocation{},
		&securityv1.RangeAllocationList{},
	)
	if err := corev1.AddToScheme(scheme); err != nil {
		return err
	}
	metav1.AddToGroupVersion(scheme, securityv1.GroupVersion)
	return nil
}

func installNonCRDQuota(scheme *apimachineryruntime.Scheme) error {
	scheme.AddKnownTypes(securityv1.GroupVersion,
		&quotav1.AppliedClusterResourceQuota{},
		&quotav1.AppliedClusterResourceQuotaList{},
	)
	if err := corev1.AddToScheme(scheme); err != nil {
		return err
	}
	metav1.AddToGroupVersion(scheme, quotav1.GroupVersion)
	return nil
}
