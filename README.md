## Intro

This SDK is used to develop the WASM Plugins for Higress in Go.

## Build on local yourself

You can also build wasm locally and copy it to a Docker image. This requires a local build environment:

Go version: >= 1.24

The following is an example of building the plugin [request-block](examples/request-block).

### step1. build wasm

```bash
cd examples/request-block
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go
```

### step2. build and push docker image

A simple Dockerfile:

```Dockerfile
FROM scratch
COPY main.wasm plugin.wasm
```

```bash
docker build -t <your_registry_hub>/request-block:1.0.0 -f <your_dockerfile> .
docker push <your_registry_hub>/request-block:1.0.0
```

## Apply WasmPlugin API

Read this [document](https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/) to learn more about wasmplugin.

Create a WasmPlugin API resource:

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  defaultConfig:
    block_urls:
    - "swagger.html"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

When the resource is applied on the Kubernetes cluster with `kubectl apply -f <your-wasm-plugin-yaml>`,
the request will be blocked if the string `swagger.html` in the url. 

```bash
curl <your_gateway_address>/api/user/swagger.html
```

```text
HTTP/1.1 403 Forbidden
date: Wed, 09 Nov 2022 12:12:32 GMT
server: istio-envoy
content-length: 0
```

## route-level & domain-level takes effect

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: request-block
  namespace: higress-system
spec:
  defaultConfig:
   # this config will take effect globally (all incoming requests not matched by rules below)
   block_urls:
   - "swagger.html"
  matchRules:
  # ingress-level takes effect
  - ingress:
    - default/foo
    # the ingress foo in namespace default will use this config
    config:
      block_bodies:
      - "foo"
  - ingress:
    - default/bar
    # the ingress bar in namespace default will use this config
    config:
      block_bodies:
      - "bar"
  # domain-level takes effect
  - domain:
    - "*.example.com"
    # if the request's domain matched, this config will be used
    config:
      block_bodies:
       - "foo"
       - "bar"
  url: oci://<your_registry_hub>/request-block:1.0.0
```

The rules will be matched in the order of configuration. If one match is found, it will stop, and the matching configuration will take effect.

## Unit Testing

For comprehensive unit testing support, see our [Test Framework Documentation](pkg/test/README.md).

## Request body buffering

Registering `ProcessRequestBody` enables complete request body processing. For a
request that needs buffering, the wrapper holds subsequent filters' headers with
`types.HeaderStopIteration` when your request header handler returns Continue.
After DATA or trailers complete the body, the wrapper invokes the body handler
once and preserves its returned action. This also works without registering a
request header handler.

Holding headers prevents a later asynchronous filter, such as a rate limiter,
from resuming Envoy's shared request buffer before this plugin has processed the
complete body. Consequently, later header handlers and their external calls run
after body processing rather than concurrently with the request upload.

Requests carrying `Expect: 100-continue`, valid HTTP Upgrade headers, or the
`CONNECT` method retain their original header action so protocol negotiation
can finish before data arrives. This exception also applies when
`RequireRequestBody()` is set: it does not cancel required body delivery or
change body processing settings. These compatibility paths retain the previous
shared-buffer behavior and do not gain the new complete-body header barrier.
If HCM handles 100 Continue locally and removes Expect before filters run, the
ordinary automatic header hold still applies.

Per-request `DontReadRequestBody()` and streaming request body processing retain
their existing behavior. Binary, encoded and WebSocket requests retain the
existing automatic opt-outs unless `RequireRequestBody()` explicitly overrides
them. A request already ended in headers is not paused just to wait for a body.

Explicit header actions, including `HeaderStopAllIterationAndBuffer` and
`HeaderStopAllIterationAndWatermark`, are preserved. The wrapper does not delay
direct `proxywasm.ResumeHttpRequest()` calls: plugins that start their own
asynchronous header work must coordinate continuation themselves. Do not resume
while still relying on the wrapper to retain an incomplete body.

For a real `transformer` → `ai-token-ratelimit` regression check, use
[`test/integration/buffered_request_body.py`](test/integration/buffered_request_body.py).
It builds the community plugins against this SDK and checks split uploads,
trailer termination, binary opt-outs and rate-limit rejection in an isolated
Higress Envoy with Redis. A complete-body fixture also checks upstream/local
100 Continue, WebSocket 101 and CONNECT 200 before sending request data, including
required-body delivery and Connection token lists.
Run it with `--help` for prerequisites and arguments;
`--baseline-sdk` additionally reproduces the failure using an unpatched checkout.
