module git.arvan.me/arvan/cli

go 1.15

require (
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/alexbrainman/sspi v0.0.0-20200928142253-2a432fede40d // indirect
	github.com/apcera/gssapi v0.0.0-20161010215902-5fb4217df13b // indirect
	github.com/aws/aws-sdk-go v1.36.7 // indirect
	github.com/containerd/continuity v0.0.0-20201208142359-180525291bb7 // indirect
	github.com/containers/image v3.0.2+incompatible // indirect
	github.com/containers/storage v1.18.2 // indirect
	github.com/docker/docker v20.10.0+incompatible // indirect
	github.com/fsouza/go-dockerclient v1.6.6 // indirect
	github.com/gonum/graph v0.0.0-20190426092945-678096d81a4b // indirect
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/buildkit v0.8.0 // indirect
	github.com/moby/term v0.0.0-20201110203204-bea5bbe245bf // indirect
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible // indirect
	github.com/openshift/library-go v0.0.0-20201211095848-8399bf6288d6
	github.com/openshift/oc v0.0.0-alpha.0.0.20201210232229-4ebfe9cad4c3
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.20.0-beta.2
	k8s.io/apimachinery v0.20.0-beta.2
	k8s.io/cli-runtime v0.20.0-beta.2
	k8s.io/client-go v0.20.0-beta.2
	k8s.io/kubectl v0.20.0-beta.2
)

replace (
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.7
	github.com/apcera/gssapi => github.com/openshift/gssapi v0.0.0-20161010215902-5fb4217df13b
	github.com/containerd/containerd => github.com/containerd/containerd v1.3.6
	github.com/containers/image => github.com/openshift/containers-image v0.0.0-20190130162819-76de87591e9d
	// Taking changes from https://github.com/moby/moby/pull/40021 to accomodate new version of golang.org/x/sys.
	// Although the PR lists c3a0a3744636069f43197eb18245aaae89f568e5 as the commit with the fixes,
	// d1d5f6476656c6aad457e2a91d3436e66b6f2251 is more suitable since it does not break fsouza/go-clientdocker,
	// yet provides the same fix.
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20191121165722-d1d5f6476656

	// Temporary prebase beta.2 pins
	github.com/openshift/api => github.com/openshift/api v0.0.0-20201119144013-9f0856e7c657
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20201119144744-148025d790a9
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20201119162840-a8387fdfa05b
	github.com/openshift/oc => github.com/arvancloud/oc v0.0.0-alpha.0.0.20201229052306-dff55bd9edf0

	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975

	k8s.io/api => k8s.io/api v0.20.0-beta.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.0-beta.2
	k8s.io/apimachinery => github.com/openshift/kubernetes-apimachinery v0.0.0-20201119164651-a0d1e1af7af8
	k8s.io/apiserver => k8s.io/apiserver v0.20.0-beta.2
	k8s.io/cli-runtime => github.com/openshift/kubernetes-cli-runtime v0.0.0-20201120205941-cafce159b165
	k8s.io/client-go => github.com/openshift/kubernetes-client-go v0.0.0-20201119165025-c1570ba06fef
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.0-beta.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.0-beta.2
	k8s.io/code-generator => k8s.io/code-generator v0.20.0-beta.2
	k8s.io/component-base => k8s.io/component-base v0.20.0-beta.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.20.0-beta.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.20.0-beta.2
	k8s.io/cri-api => k8s.io/cri-api v0.20.0-beta.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.0-beta.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.0-beta.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.0-beta.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.0-beta.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.0-beta.2
	k8s.io/kubectl => github.com/openshift/kubernetes-kubectl v0.0.0-20201120182004-53f855031220
	k8s.io/kubelet => k8s.io/kubelet v0.20.0-beta.2
	k8s.io/kubernetes => github.com/openshift/kubernetes v1.20.0-beta.2.0.20201120184952-99ac8bcc32c6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.0-beta.2
	k8s.io/metrics => k8s.io/metrics v0.20.0-beta.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.20.0-beta.2
	k8s.io/node-api => k8s.io/node-api v0.20.0-beta.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.0-beta.2
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.0-beta.2
	k8s.io/sample-controller => k8s.io/sample-controller v0.20.0-beta.2

)
