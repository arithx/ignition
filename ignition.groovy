job('ignition') {
    scm {
        git {
            remote {
                github('coreos/ignition')
                refspec('+refs/pull/*:refs/remotes/origin/pr/*')
            }
            branch('${sha1}')
        }
    }

    triggers {
        pullRequest {
            orgWhitelist('coreos')
            cron('H/5 * * * *')
            triggerPhrase('OK to test')
            onlyTriggerPhrase()
            useGitHubHooks()
            allowMembersOfWhitelistedOrgsAsAdmin()
            extensions {
                commitStatus {
                    context('Black Box Tests')
                    completedStatus('SUCCESS')
                    completedStatus('FAILURE')
                    completedStatus('PENDING')
                    completedStatus('ERROR')
                }
            }
        }
    }

    stages {
        stage('Test') {
            agent {
                label 'amd64&&docker'
            }
            steps {
                sh '''#!/bin/bash -ex

docker run --rm -e TARGET=amd64 -v "$PWD":/usr/src/myapp -w /usr/src/myapp quay.io/slowrie/ignition-builder ./build_blackbox_tests
trap 'sudo rm -rf ./bin' EXIT
sudo -E PATH=$PWD/bin/amd64:$PATH ./tests.test -test.v
'''
            }
        }
    }

    post {
        always {
            cleanWs()
        }
    }
}
