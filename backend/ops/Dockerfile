FROM cgr.dev/chainguard/wolfi-base:latest

RUN apk add --no-cache \
        aws-cli-v2 \
        bash \
        ca-certificates-bundle \
        curl \
        jq \
        postgresql-16-client

USER nonroot

CMD ["sleep", "infinity"]
