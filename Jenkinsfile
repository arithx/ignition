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

sudo chmod +x docker_build
sudo chmod +x test
sudo chmod +x build
sudo chmod +x ./coreos_test
'''
                withDockerContainer("docker pull quay.io/slowrie/ignition-builder-${GOVERSION}") {
                    sh '''#!/bin/bash

if [ "${TARGET}" == "amd64" ]; then
    export ACTION="COMPILE"
    GOARCH="${TARGET}" ./test;
elif [ "${TARGET}" == "arm64" ]; then
    export CGO_LDFLAGS="-L ${PWD}";
    GOARCH="${TARGET}" ./build;
    file "bin/${TARGET}/ignition" | egrep 'aarch64';
fi
'''
                }

                sh '''#!/bin/bash -ex

cleanup()
{
    if [ "$(cat /proc/mounts | grep /tmp/hd)" != "" ]; then
        find /tmp/hd* | xargs umount
    fi

    if [ ! -d "test.img" ]; then
        rm -rf test.img
    fi
}

PATH=$PATH:$PWD/bin/amd64
OUT=$(sudo -E PATH=$PATH find -name "*.test" -exec '{}' ';')
echo $OUT

# Run cleanup before potentially exiting
cleanup

if [ "${OUT#*FAIL}" != "$OUT" ]; then
    exit 1
fi
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
