# API Documentation

## Available APIs

The server exposes **3 main APIs** plus a health check endpoint.

---

### API 1: Get Resource History
**Endpoint:** `GET /api/history`

**Parameters:**
- `kind` (required): Resource kind (e.g., HTTPRoute, Gateway)
- `name` (required): Resource name
- `namespace` (required): Resource namespace

**Returns:** JSON array of generation and timestamp pairs

**Example Request:**
```bash
curl "http://localhost:8080/api/history?kind=HTTPRoute&name=example-route&namespace=default"
```

**Example Response:**
```json
[
  {
    "generation": 1,
    "timestamp": "2026-02-03T06:03:01Z"
  },
  {
    "generation": 2,
    "timestamp": "2026-02-03T06:10:15Z"
  }
]
```

---

### API 2: Get Specific Generation YAML
**Endpoint:** `GET /api/generation`

**Parameters:**
- `kind` (required): Resource kind (e.g., HTTPRoute, Gateway)
- `name` (required): Resource name
- `namespace` (required): Resource namespace
- `generation` (required): Generation number

**Returns:** YAML for the specified generation

**Example Request:**
```bash
curl "http://localhost:8080/api/generation?kind=HTTPRoute&name=example-route&namespace=default&generation=1"
```

**Example Response:**
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example-route
  namespace: default
  generation: 1
  creationTimestamp: "2026-02-03T06:03:01Z"
spec:
  hostnames:
  - example.com
  parentRefs:
  - name: example-gateway
    namespace: default
  rules:
  - backendRefs:
    - name: example-service
      port: 5000
    matches:
    - path:
        type: PathPrefix
        value: /
```

---

### API 3: List All Resources
**Endpoint:** `GET /api/resources`

**Parameters:** None

**Returns:** JSON array of all resource tuples (kind/name/namespace)

**Example Request:**
```bash
curl "http://localhost:8080/api/resources"
```

**Example Response:**
```json
[
  {
    "kind": "HTTPRoute",
    "name": "example-route",
    "namespace": "default"
  },
  {
    "kind": "HTTPRoute",
    "name": "example-route-2",
    "namespace": "default"
  },
  {
    "kind": "Gateway",
    "name": "example-gateway",
    "namespace": "default"
  }
]
```

---

### Health Check
**Endpoint:** `GET /health`

**Parameters:** None

**Returns:** Server health status

**Example Request:**
```bash
curl "http://localhost:8080/health"
```

**Example Response:**
```json
{
  "success": true,
  "message": "Server is healthy"
}
```

---

## Testing Examples

```bash
# 1. Check server health
curl http://localhost:8080/health

# 2. List all resources
curl http://localhost:8080/api/resources

# 3. Get history for a specific resource
curl "http://localhost:8080/api/history?kind=HTTPRoute&name=example-route&namespace=default"

# 4. Get specific generation YAML
curl "http://localhost:8080/api/generation?kind=HTTPRoute&name=example-route&namespace=default&generation=1"
```

---

## Error Responses

All endpoints return error responses in the following format:

```json
{
  "success": false,
  "error": "Error message here"
}
```

**Common HTTP Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Missing or invalid parameters
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method (only GET is supported)
- `500 Internal Server Error` - Server error

---

## Redis Storage Format

Resources are stored in Redis with the key format: `{kind}/{name}/{namespace}`

Examples:
- `HTTPRoute/example-route/default`
- `Gateway/example-gateway/default`

Each key contains a list of resource versions (most recent first), with a maximum of 100 versions per resource (configurable via `--max-changes` flag).
