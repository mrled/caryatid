#!/usr/bin/env groovy

node() {
    // TODO: Set version from a file committed to Git?
    String version = "2.0.${env.BUILD_NUMBER}"
    // TODO: Is there a way to derive this automatically?
    String projGoSubpath = "/go/src/github.com/mrled/caryatid"

    stage('Checkout from GitHub') {
        checkout scm
    }

    stage('Build, test, and buildrelease') {
        docker.image("golang:1.9-alpine").inside("-v ${pwd()}:${projGoSubpath}") {
            for (command in binaryBuildCommands) {
                sh "cd ${projGoSubpath} && go build ./..."
                sh "cd ${projGoSubpath} && go test ./..."
                sh "cd ${projGoSubpath} && go run scripts/buildrelease.go -version ${version}"
            }
        }
    }

    // TODO: Upload to GitHub automatically if we are building a commit with a version tag like vX.Y.Z(-PRERELEASETAG)?

    stage("Archive artifacts") {
        archiveArtifacts artifacts: 'release/**', fingerprint: true
    }
}
