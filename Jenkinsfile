#!/usr/bin/env groovy

node {
    // TODO: Set version from a file committed to Git?
    String version = "2.0.${env.BUILD_NUMBER}"
    // TODO: Is there a way to derive this automatically?
    String projGoSubpath = "/go/src/github.com/mrled/caryatid"

    stage('1. Checkout from GitHub') {
        checkout scm
    }

    stage('2. ... fuckin where am i') {
        sh "printf pwd: \$(pwd)"
        sh "printf contents: \$(ls -R)"
    }

    stage('3. Build, test, and buildrelease') {
        docker.image("golang:1.9-alpine").inside("-v ${pwd()}:${projGoSubpath}") {
            sh "echo projGoSubpath: ${projGoSubpath}. Contents:"
            sh "ls ${projGoSubpath}"
            sh "echo End Contents"
            sh "echo GOPATH: \$GOPATH"
            sh "echo GOROOT: \$GOROOT"
            sh "echo /go Contents:"
            sh "ls -R /go"
            sh "echo End Contents"
            sh "cd ${projGoSubpath} && go build ./..."
            sh "cd ${projGoSubpath} && go test ./..."
            sh "cd ${projGoSubpath} && go run scripts/buildrelease.go -version ${version}"
        }
    }

    // TODO: Upload to GitHub automatically if we are building a commit with a version tag like vX.Y.Z(-PRERELEASETAG)?

    stage("4. Archive artifacts") {
        archiveArtifacts artifacts: 'release/**', fingerprint: true
    }
}
