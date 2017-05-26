#!groovy

properties([
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30')),

    parameters([
        choice(name: 'GOARCH',
               choices: "amd64",
               description: 'target architecture for building binaries'),
        choice(name: 'GOVERSION',
               choices: "1.7\n1.8",
               description: 'version of golang')
    ]),

    pipelineTriggers([pollSCM('H/15 * * * *')])
])

node('amd64 && docker') {
    stage('SCM') {
        checkout scm
    }

    stage('Build & Test') {
        GOARCH = params.GOARCH
        GOVERSION = params.GOVERSION
        CGO_ENABLED = "0"

        sh 'echo export TARGET=${GOARCH} > env_vars'
        sh 'echo export GOARCH=${GOARCH} >> env_vars'
        sh 'echo export CGO_ENABLED=${CGO_ENABLED} >> env_vars'
        sh 'cat env_vars'

        sh 'sed -i "s/_GOVERSION_/${GOVERSION}/g" Dockerfile'
        sh 'docker build --rm --tag=test .'
        sh 'docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test chmod +x docker_build;'
        sh 'docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test chmod +x test;'
        sh 'docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test chmod +x build;'
        sh 'docker run --rm -e TARGET=${GOARCH} -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test sudo -E "PATH=$PATH:/go/bin:/usr/local/go/bin" ./docker_build'

        if (GOARCH=="amd64") {
            sh 'sed -i "s/_GOVERSION_/${GOVERSION}/g" coreos_test'
            sh 'sudo chmod +x ./coreos_test; sudo -E ./coreos_test'
        }
    }
}
