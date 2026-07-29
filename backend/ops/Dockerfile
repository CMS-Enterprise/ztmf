FROM cgr.dev/chainguard/wolfi-base:latest

RUN apk add --no-cache \
        aws-cli-v2 \
        bash \
        ca-certificates-bundle \
        coreutils \
        curl \
        jq \
        postgresql-16-client

# RDS global trust bundle, so psql/pg_dump can run with sslmode=verify-full
# instead of merely encrypting. Not present in the distro CA bundle.
ADD --chmod=0444 https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem \
    /etc/ssl/certs/rds-global-bundle.pem

COPY --chmod=0555 dump-db.sh /usr/local/bin/dump-db.sh
COPY --chmod=0555 check-dump-age.sh /usr/local/bin/check-dump-age.sh

USER nonroot

# Default is the interactive shell target used by scripts/db-tunnel.sh. The
# scheduled monthly backup task overrides this with dump-db.sh.
CMD ["sleep", "infinity"]
