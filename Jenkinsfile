#!groovy

properties([
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30')),

    pipelineTriggers([pollSCM('H/15 * * * *')])
])

def test_ignition(ARCH, GOVERSION)
{
    node("${ARCH} && docker") {
        def CGO = (ARCH == 'arm64') ? 1 : 0
        withEnv(["TARGET=${ARCH}", "CGO_ENABLED=${CGO}",
                 "GOARCH=${ARCH}", "GOVERSION=${GOVERSION}"]) {
            stage("SCM $GOVERSION") {
                checkout scm
            }

            stage("Build & Test $GOVERSION") {
                sh '''#!/bin/bash -ex

sed -i "s/_GOVERSION_/$GOVERSION/g" Dockerfile
docker build --rm --tag=test .
docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test chmod +x docker_build;
docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test chmod +x test;
docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test chmod +x build;
docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test sudo -E "PATH=$PATH:/go/bin:/usr/local/go/bin" ./docker_build

sudo chmod +x ./coreos_test; sudo -E ./coreos_test
'''
            }
        }
    }
}


def archs = ['amd64']
def govers = ['1.7', '1.8']

for (String arch : archs) {
    for (String gover : govers) {
        test_ignition(arch, gover)
    }
}
