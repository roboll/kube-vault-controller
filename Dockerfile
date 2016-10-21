FROM alpine:3.4

RUN apk add --no-cache --update ca-certificates

ADD kube-vault-controller /kube-vault-controller
ENTRYPOINT ["/kube-vault-controller"]
