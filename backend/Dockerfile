FROM golang:1.25.14-bookworm AS builder

WORKDIR /src

RUN update-ca-certificates && openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -nodes -sha256 -days 3650 \ 
  -subj "/C=US/ST=Maryland/L=Baltimore/O=Centers for Medicare and Medicaid Services/OU=OIT\ISPG/CN=us-east-1.elb.amazonaws.com" \
  -addext "subjectAltName=DNS:*.us-east-1.elb.amazonaws.com"

COPY ./go.mod /src/
COPY ./go.sum /src/
COPY ./cmd/api /src/cmd/api
COPY ./internal/ /src/internal/

RUN go build -o ./ ./cmd/api/...

FROM gcr.io/distroless/base-debian12 AS runtime

WORKDIR /src

COPY --from=builder /src/api /usr/local/bin/ztmfapi
COPY --from=builder /src/*.pem /src/

# Test image: runtime plus the empire seed, so an ephemeral task can self-seed
# with DB_POPULATE and no host mount. Build with --target test.
FROM runtime AS test

COPY ./_test_data_empire.sql /app/_test_data_empire.sql

# Default target stays the seedless runtime image.
FROM runtime
