#!groovy

properties([
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30')),

    pipelineTriggers([pollSCM('H/15 * * * *')])
])

def test_ignition(GOVERSION)
{
    node('amd64 && docker') {
        withEnv(["TARGET=amd64", "CGO_ENABLED=0",
                 "GOARCH=amd64", "GOVERSION=${GOVERSION}"]) {
            stage("SCM $GOVERSION") {
                checkout scm
            }

            stage("Build & Test $GOVERSION") {
                def GOARCH = "amd64"
                def CGO_ENABLED = "0"
                if (GOARCH=="arm64") {
                    CGO_ENABLED = "1"
                }

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

test_ignition("1.7")
test_ignition("1.8")
