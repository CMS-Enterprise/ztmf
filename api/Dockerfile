FROM --platform=linux/amd64 golang:1.22.0-bookworm as builder

WORKDIR /api

RUN openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -nodes -sha256 -days 3650 \ 
  -subj "/C=US/ST=Maryland/L=Baltimore/O=Centers for Medicare and Medicaid Services/OU=OIT\ISPG/CN=us-east-1.elb.amazonaws.com" \
  -addext "subjectAltName=DNS:*.us-east-1.elb.amazonaws.com"

COPY ./go.mod /api/
COPY ./go.sum /api/
COPY ./engine/ /api/engine/
COPY ./config/ /api/config/
COPY ./db/ /api/db/
COPY main.go /api/

RUN go install ./...

FROM --platform=linux/amd64 scratch

WORKDIR /api

COPY --from=builder /go/bin/api /usr/local/bin/ztmfapi
COPY --from=builder /api/*.pem /api/
COPY --from=builder /lib/x86_64-linux-gnu/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=builder /usr/lib/x86_64-linux-gnu/libc.so.6 /usr/lib/x86_64-linux-gnu/libc.so.6
