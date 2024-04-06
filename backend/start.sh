#!/bin/bash

export DB_CREDS=$(aws --profile ztmf-dev secretsmanager get-secret-value --secret-id $DB_CREDS_SECRET_ID --query 'SecretString' --output text)
export DB_NAME="ztmf"
go run ./...
