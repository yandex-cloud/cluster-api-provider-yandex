FROM golang:1.22 as build

ENV CGO_ENABLED=0

WORKDIR /workspace

RUN apt update && apt install ca-certificates
RUN update-ca-certificates

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
# Copy only go sources
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY internal/ internal/

# Build
ARG ARCH
RUN GOARCH=${ARCH} \
    go build -o manager .

FROM scratch
COPY --from=build /workspace/manager /bin/manager
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /

ENTRYPOINT ["/bin/manager"]
