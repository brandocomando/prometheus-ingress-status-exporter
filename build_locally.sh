#!/bin/bash

if [[ $GOPATH != "" ]]
then
  DIR=`pwd`
  TIMESTAMP=`date +%s`
  mkdir -p $GOPATH/src/ingress-status-exporter-$TIMESTAMP
  cp src/* $GOPATH/src/ingress-status-exporter-$TIMESTAMP/
  cd $GOPATH/src/ingress-status-exporter-$TIMESTAMP/
  go get ./
  go install ./
  go build -o $DIR/ingress-status-exporter
  rm -r $GOPATH/src/ingress-status-exporter-$TIMESTAMP
  echo "Build Sucessful, try it out ./ingress-status-exporter -h"
else
  echo "GOPATH not set, not building"
fi
