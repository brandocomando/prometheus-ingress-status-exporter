#!/bin/bash

if [[ $GOPATH != "" ]]
then
  DIR=`pwd`
  mkdir -p $GOPATH/src/ingress-status-exporter
  cp src/* $GOPATH/src/ingress-status-exporter/
  cd $GOPATH/src/ingress-status-exporter/
  go get ./
  go install ./
  go build -o $DIR/ingress-status-exporter
  rm -r $GOPATH/src/ingress-status-exporter
  echo "Build Sucessful, try it out ./ingress-status-exporter -h"
else
  echo "GOPATH not set, not building"
fi
