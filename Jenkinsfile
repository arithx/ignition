#!groovy

properties([
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30')),

    parameters([
        choice(name: 'GOARCH',
               choices: "amd64\narm64",
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
        CGO_ENABLED = (${params.GOARCH}=="arm64") ? 1 : 0
        sh 'docker run --rm -e GOARCH=${params.GOARCH} -e CGO_ENABLED=${CGO_ENABLED} -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.8.1 ./test'
    }
}
