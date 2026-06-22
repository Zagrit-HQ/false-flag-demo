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

target "api" {
  context    = "."
  dockerfile = "infra/Dockerfile"
  args = {
    SERVICE = "falseflag-api"
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
  tags      = ["${REGISTRY}/api:${VERSION}"]
}

target "proxy" {
  context    = "."
  dockerfile = "infra/Dockerfile"
  args = {
    SERVICE = "falseflag-proxy"
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
  tags      = ["${REGISTRY}/proxy:${VERSION}"]
}

target "operator" {
  context    = "."
  dockerfile = "infra/Dockerfile"
  args = {
    SERVICE = "falseflag-operator"
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
  tags      = ["${REGISTRY}/operator:${VERSION}"]
}

target "mcp" {
  context    = "."
  dockerfile = "infra/Dockerfile"
  args = {
    SERVICE = "falseflag-mcp"
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
  tags      = ["${REGISTRY}/mcp:${VERSION}"]
}

target "loadgen" {
  context    = "."
  dockerfile = "infra/Dockerfile"
  args = {
    SERVICE = "falseflag-loadgen"
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
  tags      = ["${REGISTRY}/loadgen:${VERSION}"]
}

target "dashboard" {
  context    = "."
  dockerfile = "infra/Dockerfile.dashboard"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/dashboard:${VERSION}"]
}
