FROM --platform=$BUILDPLATFORM golang:1.19.0-bullseye as builder
WORKDIR /go/src/github.com/kubernetes-sigs/aws-file-cache-csi-driver
ADD . .
RUN make

FROM amazonlinux:2 AS linux-amazon
RUN yum update -y
RUN yum install util-linux libyaml -y \
    && amazon-linux-extras install -y lustre

COPY --from=builder /go/src/github.com/kubernetes-sigs/aws-file-cache-csi-driver/bin/aws-file-cache-csi-driver /bin/aws-file-cache-csi-driver

ENTRYPOINT ["/bin/aws-file-cache-csi-driver"]
