#!/usr/bin/env groovy

pipeline {
    agent any

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        stage('2. ... fuckin where am i') {
            steps {
                sh "printf pwd: \$(pwd)"
                sh "printf contents: \$(ls -R)"
            }
        }
        stage('3. Build, test, and buildrelease') {
            steps {
                docker.image("golang:1.9-alpine").inside("-v ${pwd()}:/go/src/github.com/mrled/caryatid") {
                    sh "echo projGoSubpath: /go/src/github.com/mrled/caryatid. Contents:"
                    sh "ls /go/src/github.com/mrled/caryatid"
                    sh "echo End Contents"
                    sh "echo GOPATH: \$GOPATH"
                    sh "echo GOROOT: \$GOROOT"
                    sh "echo /go Contents:"
                    sh "ls -R /go"
                    sh "echo End Contents"
                    sh "cd /go/src/github.com/mrled/caryatid && go build ./..."
                    sh "cd /go/src/github.com/mrled/caryatid && go test ./..."
                    sh "cd /go/src/github.com/mrled/caryatid && go run scripts/buildrelease.go -version 2.0.${env.BUILD_NUMBER}"
                }
            }
        }
        stage("4. Archive artifacts") {
            steps {
                archiveArtifacts artifacts: 'release/**', fingerprint: true
            }
        }
    }
}
