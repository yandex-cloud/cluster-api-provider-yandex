FROM golang:1.22 as build

ENV CGO_ENABLED=0

WORKDIR /workspace

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

FROM ubuntu:latest
COPY --from=build /workspace/manager /bin/manager
WORKDIR /

ENTRYPOINT ["/bin/manager"]
