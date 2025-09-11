# WASM Go SDK

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

## MCP (Model Context Protocol) Support

This SDK provides comprehensive support for MCP (Model Context Protocol), including the latest features from MCP protocol version 2025-06-18.

### Output Schema Support

Output schema allows tools to define the structure of their response data, enabling better type safety and validation for MCP clients.

#### Example: REST Tool with Output Schema

```yaml
server:
  name: weather-api
  config:
    apiKey: "your-api-key"
tools:
  - name: get_weather
    description: "Get current weather information"
    args:
      - name: city
        type: string
        description: "City name"
        required: true
      - name: units
        type: string
        description: "Temperature units"
        enum: ["celsius", "fahrenheit"]
        default: "celsius"
    outputSchema:
      type: object
      properties:
        temperature:
          type: number
          description: "Current temperature"
        humidity:
          type: number
          description: "Humidity percentage"
        condition:
          type: string
          description: "Weather condition"
        city:
          type: string
          description: "City name"
      required: ["temperature", "condition", "city"]
    requestTemplate:
      url: "https://api.weather.com/v3/weather?city={{.args.city}}&units={{.args.units}}&key={{.config.apiKey}}"
      method: "GET"
    responseTemplate:
      body: "{{.}}"
```

#### Example: Direct Response Tool with Output Schema

```yaml
server:
  name: calculator
tools:
  - name: add_numbers
    description: "Add two numbers"
    args:
      - name: a
        type: number
        description: "First number"
        required: true
      - name: b
        type: number
        description: "Second number"
        required: true
    outputSchema:
      type: object
      properties:
        result:
          type: number
          description: "Sum of the two numbers"
        operation:
          type: string
          description: "Operation performed"
      required: ["result", "operation"]
    responseTemplate:
      body: |
        {
          "result": {{add .args.a .args.b}},
          "operation": "addition"
        }
### MCP Protocol Versions

The SDK supports multiple MCP protocol versions:

- **2024-11-05**: Initial MCP specification
- **2025-03-26**: Enhanced tool capabilities
- **2025-06-18**: Output schema support (latest)

### Features

- ✅ REST API integration with template support
- ✅ Structured data handling with JSON validation
- ✅ Output schema for type-safe responses
- ✅ Security scheme support (HTTP Basic, Bearer, API Key)
- ✅ Tool composition and toolsets
- ✅ Comprehensive error handling
- ✅ Full backward compatibility

### Migration Guide: Adding Output Schema to Existing Tools

If you have existing MCP tools that don't use output schema, you can easily add output schema support to provide better type safety and validation for clients.

#### Step 1: Identify Your Tool's Response Structure

First, analyze what your tool returns. For example, if your tool returns weather data:

```yaml
# Before: Tool without output schema
tools:
  - name: get_weather
    description: "Get current weather"
    args:
      - name: city
        type: string
        required: true
    requestTemplate:
      url: "https://api.weather.com/current?city={{.args.city}}"
      method: "GET"
    responseTemplate:
      body: "{{.}}"
```

#### Step 2: Define Output Schema Based on Response

Add an `outputSchema` field that matches your tool's actual response structure:

```yaml
# After: Tool with output schema
tools:
  - name: get_weather
    description: "Get current weather"
    args:
      - name: city
        type: string
        required: true
    outputSchema: # ← Add this field
      type: object
      properties:
        temperature:
          type: number
          description: "Temperature in Celsius"
        humidity:
          type: number
          description: "Humidity percentage"
        condition:
          type: string
          description: "Weather condition (sunny, cloudy, rainy)"
        city:
          type: string
          description: "City name"
      required: ["temperature", "condition", "city"] # Specify required fields
    requestTemplate:
      url: "https://api.weather.com/current?city={{.args.city}}"
      method: "GET"
    responseTemplate:
      body: "{{.}}"
```

#### Step 3: Handle Different Response Types

##### For JSON API Responses

If your tool calls a JSON API, the response body will be automatically validated:

```yaml
tools:
  - name: get_user_profile
    description: "Get user profile information"
    args:
      - name: user_id
        type: string
        required: true
    outputSchema:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
        created_at:
          type: string
          format: date-time
      required: ["id", "name"]
    requestTemplate:
      url: "https://api.example.com/users/{{.args.user_id}}"
      method: "GET"
    responseTemplate:
      body: "{{.}}" # Raw JSON response
```

##### For Template-Based Responses

For direct response tools with templates, ensure your template generates valid JSON:

```yaml
tools:
  - name: calculate_bmi
    description: "Calculate BMI from height and weight"
    args:
      - name: height
        type: number
        description: "Height in meters"
        required: true
      - name: weight
        type: number
        description: "Weight in kilograms"
        required: true
    outputSchema:
      type: object
      properties:
        bmi:
          type: number
          description: "Body Mass Index"
        category:
          type: string
          enum: ["underweight", "normal", "overweight", "obese"]
        interpretation:
          type: string
      required: ["bmi", "category"]
    responseTemplate:
      body: |
        {
          "bmi": {{div .args.weight (mul .args.height .args.height)}},
          "category": "{{if lt (div .args.weight (mul .args.height .args.height)) 18.5}}underweight{{else if lt (div .args.weight (mul .args.height .args.height)) 25}}normal{{else if lt (div .args.weight (mul .args.height .args.height)) 30}}overweight{{else}}obese{{end}}",
          "interpretation": "BMI calculated from height {{.args.height}}m and weight {{.args.weight}}kg"
        }
```

#### Step 4: Validate Your Schema

Test your tool to ensure the output schema matches the actual response:

```bash
# Test the tool
curl -X POST http://your-mcp-server/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "get_weather",
    "arguments": {"city": "New York"}
  }'
```

#### Step 5: Gradual Migration Strategy

You can migrate tools incrementally:

1. **Start with new tools**: Add output schema to all new tools
2. **Migrate high-traffic tools first**: Focus on tools used frequently
3. **Test thoroughly**: Ensure schema matches actual responses
4. **Update client code**: Modify MCP clients to use the schema information

#### Important Notes

- **Backward Compatibility**: Existing tools without output schema continue to work
- **Optional Implementation**: Output schema is optional - tools can implement it when ready
- **JSON Validation**: The SDK automatically validates JSON responses against the schema
- **Error Handling**: Invalid responses are handled gracefully with appropriate error messages

For more detailed MCP documentation, see [pkg/mcp/validator/README.md](pkg/mcp/validator/README.md).
