## Intro

This SDK is used to develop the WASM Plugins for Higress in Go.

## Build on local yourself

You can also build wasm locally and copy it to a Docker image. This requires a local build environment:

Prerequisites:

- Go version: >= 1.24
- Binaryen `wasm-opt` version 130 available in `PATH`

The following is an example of building the plugin [request-block](examples/request-block).

### step1. build wasm

```bash
cd examples/request-block
make
```

The default build uses `scripts/build-wasm.sh`. It compiles a stripped Go Wasm
release artifact and then runs `wasm-opt -Oz --enable-bulk-memory`. The
equivalent commands are:

```bash
GOOS=wasip1 GOARCH=wasm go build -trimpath -buildmode=c-shared \
  -ldflags='-s -w -buildid=' -o main.unoptimized.wasm .
wasm-opt main.unoptimized.wasm -Oz --enable-bulk-memory -o main.wasm
rm -f main.unoptimized.wasm
```

Use `make build-debug` when an unstripped, non-Binaryen artifact is needed for
diagnostics. `WASM_OPT=/path/to/wasm-opt make` can be used when Binaryen is not
installed in `PATH`.

### Opt-in lightweight HTTP callouts

The existing `pkg/wrapper` HTTP client keeps its `net/http.Header` API by
default for source compatibility. Plugins that want to avoid linking Go's full
HTTP stack can import the additive lightweight package:

```go
import "github.com/higress-group/wasm-go/pkg/wrapper/httpcall"
```

Build those plugins with `wasm_lite_http` so the legacy HTTP implementation in
`pkg/wrapper` is excluded:

```bash
GO_BUILD_TAGS=wasm_lite_http ../../scripts/build-wasm.sh -o main.wasm .
```

`GO_BUILD_TAGS` is passed to Go as `-tags`; an equivalent direct invocation is
`go build -tags=wasm_lite_http ...`.

The build tag only selects the HTTP API implementation. Binaryen `-Oz` remains
the default post-link optimization in both modes. Existing plugins that do not
set the tag retain the original `wrapper.HttpClient` and `net/http.Header` API.

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
