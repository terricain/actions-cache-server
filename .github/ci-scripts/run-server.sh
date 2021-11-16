#!/usr/bin/env bash

storage="$1"
database="$2"

case $storage in
s3)
  localstack_encoded=$(python -c "import urllib; print urllib.quote('''$LOCALSTACK''')")
  STORAGE_S3="s3://some-bucket/prefix/?forces3path=1&endpoint=${localstack_encoded}"
  export STORAGE_S3

  export AWS_ACCESS_KEY_ID=test
  export AWS_SECRET_ACCESS_KEY=test
  export AWS_DEFAULT_REGION=eu-west-2
  aws --endpoint-url="$LOCALSTACK" s3api create-bucket --bucket some-bucket --region eu-west-2 --create-bucket-configuration LocationConstraint=eu-west-2
  ;;
disk)
  STORAGE_DISK=$(mktemp -d)
  export STORAGE_DISK
  ;;
*)
  echo "Unknown storage backend"
  exit 1
  ;;
esac

case $database in
sqlite)
  DB_SQLITE=$(mktemp --suffix .db)
  export DB_SQLITE
  ;;
*)
  echo "Unknown database backend"
  exit 1
  ;;
esac

env
nohup ./actions-cache-server --debug --listen-address=0.0.0.0:8080 > log.txt 2>&1 &
echo $! > pid.txt

sleep 2