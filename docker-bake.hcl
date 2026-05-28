// FalseFlag docker-bake configuration. The conference demo uses this to
// build every image with Depot's remote BuildKit so the per-image work
// runs in parallel. Local `docker buildx bake` works too.

variable "VERSION" {
  default = "dev"
}

variable "COMMIT" {
  default = "unknown"
}

group "default" {
  targets = ["api", "proxy", "operator", "mcp", "dashboard"]
}

group "go-services" {
  targets = ["api", "proxy", "operator", "mcp", "loadgen"]
}

target "service-base" {
  context    = "."
  dockerfile = "infra/Dockerfile"
  args = {
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
  }
  platforms = ["linux/amd64", "linux/arm64"]
}

target "api-meta" {}
target "api" {
  inherits = ["service-base", "api-meta"]
  tags     = ["ghcr.io/depot/falseflag/api:dev"]
  args = {
    SERVICE = "falseflag-api"
  }
}

target "proxy-meta" {}
target "proxy" {
  inherits = ["service-base", "proxy-meta"]
  tags     = ["ghcr.io/depot/falseflag/proxy:dev"]
  args = {
    SERVICE = "falseflag-proxy"
  }
}

target "operator-meta" {}
target "operator" {
  inherits = ["service-base", "operator-meta"]
  tags     = ["ghcr.io/depot/falseflag/operator:dev"]
  args = {
    SERVICE = "falseflag-operator"
  }
}

target "mcp-meta" {}
target "mcp" {
  inherits = ["service-base", "mcp-meta"]
  tags     = ["ghcr.io/depot/falseflag/mcp:dev"]
  args = {
    SERVICE = "falseflag-mcp"
  }
}

target "loadgen-meta" {}
target "loadgen" {
  inherits = ["service-base", "loadgen-meta"]
  tags     = ["ghcr.io/depot/falseflag/loadgen:dev"]
  args = {
    SERVICE = "falseflag-loadgen"
  }
}

target "dashboard-meta" {}
target "dashboard" {
  context    = "."
  dockerfile = "infra/Dockerfile.dashboard"
  tags       = ["ghcr.io/depot/falseflag/dashboard:dev"]
  platforms  = ["linux/amd64", "linux/arm64"]
}
