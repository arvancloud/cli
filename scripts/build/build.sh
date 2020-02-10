#!/bin/bash
GOEXEC=${GOEXEC:-go}

if [ -z "$1" ]
  then
    exit 1
    echo -e "No output is set.\nrun script with output file in first arg\n Example: build.sh ~/go/bin/arvan"
fi

OUTPUT=$1

BUILD_TAGS="$BUILD_TAGS include_gcs include_oss containers_image_openpgp gssapi"

LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/oc/clusterup.defaultImageStreams=centos7"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/cmd/util/variable.DefaultImagePrefix=openshift/origin"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/version.majorFromGit=3"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/version.minorFromGit=11+"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/version.versionFromGit=v3.11.0+4f75300-166-dirty"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/version.commitFromGit=4f75300"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/pkg/version.buildDate=2019-05-22T10:27:08Z"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/version.gitMajor=1"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/version.gitMinor=11+"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/version.gitCommit=d4cacc0"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/version.gitVersion=v1.11.0+d4cacc0"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/version.buildDate=2019-05-22T10:27:08Z"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/kubernetes/pkg/version.gitTreeState=clean"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/client-go/pkg/version.gitMajor=1"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/client-go/pkg/version.gitMinor=11+"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/client-go/pkg/version.gitCommit=d4cacc0"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/client-go/pkg/version.gitVersion=v1.11.0+d4cacc0"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/client-go/pkg/version.buildDate=2019-05-22T10:27:08Z"
LDFLAGS="$LDFLAGS -X github.com/openshift/origin/vendor/k8s.io/client-go/pkg/version.gitTreeState=clean"

set -ex

$GOEXEC build -tags "$BUILD_TAGS" "-ldflags=$LDFLAGS" -o "$OUTPUT" cmd/arvan/arvan.go