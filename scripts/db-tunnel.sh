#!/usr/bin/env bash
# Launch an on-demand Fargate ops task and either drop a shell into it (psql
# inside the container) or open an SSM port-forward to Aurora (psql locally).
# Replaces the EC2 bastion. See infrastructure/README.md for context.
#
# Usage:
#   scripts/db-tunnel.sh <dev|prod> --shell
#   scripts/db-tunnel.sh <dev|prod> --forward [LOCAL_PORT]   # default 15432
#
# Prereqs:
#   - AWS credentials set for the target account (however you manage them:
#     aws-vault, AWS SSO, env vars, instance profile). The script does not
#     set AWS_PROFILE; whatever the calling shell resolves is used.
#   - Session Manager Plugin on PATH.
#   - jq on PATH.

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <dev|prod> --shell | --forward [LOCAL_PORT]" >&2
  exit 2
fi

ENV="$1"; shift
MODE="$1"; shift
LOCAL_PORT="${1:-15432}"

case "$ENV" in dev|prod) ;; *) echo "env must be dev or prod" >&2; exit 2 ;; esac
case "$MODE" in --shell|--forward) ;; *) echo "mode must be --shell or --forward" >&2; exit 2 ;; esac

CLUSTER="ztmf"
TASK_DEF="ops"
CONTAINER="ztmfops"

echo "[1/5] resolving network config in $ENV..."
VPC_ID=$(aws ec2 describe-vpcs \
  --filters "Name=tag:Name,Values=ztmf-east-${ENV}" \
  --query 'Vpcs[0].VpcId' --output text)
if [[ -z "$VPC_ID" || "$VPC_ID" == "None" ]]; then
  echo "could not find VPC tagged ztmf-east-${ENV}. Are your AWS credentials pointed at the $ENV account?" >&2
  exit 1
fi
SUBNETS=$(aws ec2 describe-subnets \
  --filters "Name=vpc-id,Values=${VPC_ID}" "Name=tag:use,Values=private" \
  --query 'Subnets[*].SubnetId' --output text | tr '\t' ',')
SG=$(aws ec2 describe-security-groups \
  --filters "Name=vpc-id,Values=${VPC_ID}" "Name=group-name,Values=ztmf_ops_task" \
  --query 'SecurityGroups[0].GroupId' --output text)
if [[ -z "$SUBNETS" || -z "$SG" || "$SG" == "None" ]]; then
  echo "failed to resolve subnets/SG. Is the ops task deployed in $ENV?" >&2
  exit 1
fi

echo "[2/5] launching ops task..."
TASK_ARN=$(aws ecs run-task \
  --cluster "$CLUSTER" \
  --task-definition "$TASK_DEF" \
  --enable-execute-command \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SG}],assignPublicIp=DISABLED}" \
  --started-by "${USER}-db-tunnel" \
  --query 'tasks[0].taskArn' --output text)
TASK_ID=$(basename "$TASK_ARN")
echo "    task: $TASK_ID"

cleanup() {
  echo
  echo "stopping task..."
  aws ecs stop-task --cluster "$CLUSTER" --task "$TASK_ARN" \
    --reason "db-tunnel session ended" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[3/5] waiting for task RUNNING + ExecuteCommandAgent ready (~30-60s)..."
for _ in $(seq 1 60); do
  sleep 3
  JSON=$(aws ecs describe-tasks --cluster "$CLUSTER" --tasks "$TASK_ARN")
  STATUS=$(echo "$JSON" | jq -r '.tasks[0].lastStatus')
  AGENT=$(echo "$JSON" | jq -r '.tasks[0].containers[0].managedAgents[]? | select(.name=="ExecuteCommandAgent") | .lastStatus' || true)
  if [[ "$STATUS" == "RUNNING" && "$AGENT" == "RUNNING" ]]; then break; fi
  if [[ "$STATUS" == "STOPPED" ]]; then
    REASON=$(echo "$JSON" | jq -r '.tasks[0].stoppedReason // "unknown"')
    echo "task stopped before ready: $REASON" >&2
    exit 1
  fi
done
if [[ "$STATUS" != "RUNNING" || "$AGENT" != "RUNNING" ]]; then
  echo "timed out waiting for task (status=$STATUS, agent=$AGENT)" >&2
  exit 1
fi

if [[ "$MODE" == "--shell" ]]; then
  echo "[4/5] opening interactive shell..."
  echo "[5/5] inside container: use \$DB_ENDPOINT, \$DB_SECRET_ID, etc."
  aws ecs execute-command \
    --cluster "$CLUSTER" \
    --task "$TASK_ARN" \
    --container "$CONTAINER" \
    --interactive \
    --command /bin/bash
else
  echo "[4/5] resolving Aurora endpoint..."
  AURORA_HOST=$(aws rds describe-db-clusters \
    --db-cluster-identifier ztmf \
    --query 'DBClusters[0].Endpoint' --output text)
  RUNTIME_ID=$(echo "$JSON" | jq -r '.tasks[0].containers[0].runtimeId')
  TARGET="ecs:${CLUSTER}_${TASK_ID}_${RUNTIME_ID}"

  echo "[5/5] forwarding ${AURORA_HOST}:5432 to localhost:${LOCAL_PORT}"
  echo "    connect with: psql -h localhost -p ${LOCAL_PORT} -d ztmf"
  echo "    (supply -U <user> or check the RDS master secret for credentials)"
  echo "    Ctrl-C to end."
  aws ssm start-session \
    --target "$TARGET" \
    --document-name AWS-StartPortForwardingSessionToRemoteHost \
    --parameters "host=${AURORA_HOST},portNumber=5432,localPortNumber=${LOCAL_PORT}"
fi
