FROM cgr.dev/chainguard/go:1.22@sha256:d1dd86fbdfd50c231dc0f8fd9768fbbc31df25baed17e25aab76d24a503a7d44 AS builder

WORKDIR /app

# install and cache dependencies
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# build with cache
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -o /bin/openfga ./cmd/openfga

FROM cgr.dev/chainguard/static@sha256:5e9c88174a28c259c349f308dd661a6ec61ed5f8c72ecfaefb46cceb811b55a1

EXPOSE 8081
EXPOSE 8080
EXPOSE 3000

COPY --from=builder /bin/openfga /openfga

ENTRYPOINT ["/openfga"]
