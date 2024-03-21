#!/bin/bash

export DB_CREDS=$(aws --profile ztmf-dev secretsmanager get-secret-value --secret-id 'rds!cluster-dafb0010-f3b0-4a7f-b415-06c36b769dc0' --query 'SecretString' --output text)
export DB_NAME="ztmf"
export DB_ENDPOINT="localhost"
go run ./...
