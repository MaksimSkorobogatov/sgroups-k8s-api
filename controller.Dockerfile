FROM golang:1.25 AS builder
WORKDIR /workspace
COPY --from=sgroups-proto /. /workspace/sgroups-proto/
COPY . sgroups-k8s-api/
WORKDIR /workspace/sgroups-k8s-api
RUN CGO_ENABLED=0 go build -o /sgroups-k8s-controller ./cmd/sgroups-k8s-controller

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /sgroups-k8s-controller /sgroups-k8s-controller
USER 65534:65534
ENTRYPOINT ["/sgroups-k8s-controller"]
