#!groovy

properties([
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30')),

    parameters([
        choice(name: 'GOARCH',
               choices: "amd64\narm64",
               description: 'target architecture for building binaries'),
        choice(name: 'GOVERSION',
               choices: "1.7.6\n1.8.2",
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
        CGO_ENABLED = (GOARCH=="arm64") ? 1 : 0
        sh 'sed -i "s/_GOVERSION_/${GOVERSION}/g" Dockerfile'
        sh 'docker build --rm --tag=test .'
        sh 'docker run --rm -e GOARCH=${GOARCH} -e CGO_ENABLED=${CGO_ENABLED} --privileged -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp test /bin/bash -c "chmod +x docker_build; chmod +x test; chmod +x build; echo \"\" | sudo -S ./docker_build"'
    }
}
