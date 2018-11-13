#!/bin/bash

echo "Attempting to create credentials for minio.."

SECRET_KEY=$(head -c 12 /dev/urandom | shasum| cut -d' ' -f1)
echo -n "$SECRET_KEY" | docker secret create s3-secret-key -
if [ $? = 0 ];
then
  echo "[Credentials] s3-secret-key : $SECRET_KEY"
else
  echo "[Credentials]\n s3-secret-key already exist, not creating"
fi

ACCESS_KEY=$(head -c 12 /dev/urandom | shasum| cut -d' ' -f1)
echo -n "$ACCESS_KEY" | docker secret create s3-access-key -
if [ $? = 0 ];
then
  echo "[Credentials] s3-access-key : $SECRET_KEY"
else
  echo "[Credentials]\n s3-access-key already exist, not creating"
fi

docker service rm minio

docker service create --constraint="node.role==manager" \
 --name minio \
 --detach=true --network func_functions \
 --secret s3-access-key \
 --secret s3-secret-key \
 --env MINIO_SECRET_KEY_FILE=s3-secret-key \
 --env MINIO_ACCESS_KEY_FILE=s3-access-key \
minio/minio:latest server /export


faas-cli deploy -f stack.yml


