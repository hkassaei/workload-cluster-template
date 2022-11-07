#!/bin/bash

# check for tag
if [ $# -eq 0 ]
  then 
    echo "Error: specify tag, e.g. v1.0.0"
    exit
fi

IMAGE_NAME="ghcr.io/hkassaei/edc-demo-app"
IMAGE_TAG="$1"

echo "Building Docker image ${IMAGE_NAME}:${IMAGE_TAG}"
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

# if you need to authenticate, run docker login ghcr.io -u USERNAME and enter PAT on the following prompt
echo "pushing image to ghcr"
docker push ${IMAGE_NAME}:${IMAGE_TAG}