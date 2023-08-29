FROM --platform=$BUILDPLATFORM golang:1.18 as builder
WORKDIR /workspace

COPY . .
ENV GOPROXY="https://goproxy.cn,direct"
RUN go mod download

ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

FROM ubuntu:22.10

WORKDIR /vanus-cloud

ARG git_commit

COPY --from=builder /workspace/bin/stat /vanus-cloud/bin/stat

# install ca-certificates
RUN apt update -y
RUN apt upgrade -y
RUN apt install -y ca-certificates
RUN update-ca-certificates

ENV GIT_HASH=${git_commit}

ENTRYPOINT ["bin/stat"]

EXPOSE 80

