// FalseFlag docker-bake configuration. The conference demo uses this to
// build every image with Depot's remote BuildKit so the per-image work
// runs in parallel. Local `docker buildx bake` works too.

variable "VERSION" {
  default = "dev"
}

variable "COMMIT" {
  default = "unknown"
}

variable "REGISTRY" {
  default = "ghcr.io/depot/falseflag"
}

group "default" {
  targets = ["api", "proxy", "operator", "mcp", "dashboard"]
}

group "go-services" {
  targets = ["api", "proxy", "operator", "mcp", "loadgen"]
}

target "service-base" {
  context    = ".."
  dockerfile = "infra/Dockerfile"
  args = {
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
}

target "api" {
  inherits = ["service-base"]
  args = {
    SERVICE = "falseflag-api"
  }
  tags = ["${REGISTRY}/api:${VERSION}"]
}

target "proxy" {
  inherits = ["service-base"]
  args = {
    SERVICE = "falseflag-proxy"
  }
  tags = ["${REGISTRY}/proxy:${VERSION}"]
}

target "operator" {
  inherits = ["service-base"]
  args = {
    SERVICE = "falseflag-operator"
  }
  tags = ["${REGISTRY}/operator:${VERSION}"]
}

target "mcp" {
  inherits = ["service-base"]
  args = {
    SERVICE = "falseflag-mcp"
  }
  tags = ["${REGISTRY}/mcp:${VERSION}"]
}

target "loadgen" {
  inherits = ["service-base"]
  args = {
    SERVICE = "falseflag-loadgen"
  }
  tags = ["${REGISTRY}/loadgen:${VERSION}"]
}

target "dashboard" {
  context    = ".."
  dockerfile = "infra/Dockerfile.dashboard"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/dashboard:${VERSION}"]
}
