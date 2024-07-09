#!/bin/bash

SCENARIO=$1
SUMMARY_EXPORT_DIRECTORY=$2
SUMMARY_EXPORT_FILE_NAME=$3

docker run --rm --network=host -v ${SUMMARY_EXPORT_DIRECTORY}:/results -i docker.io/grafana/k6 run - --summary-export=/results/${SUMMARY_EXPORT_FILE_NAME} --vus 2 --duration 30s < ${SCENARIO}
