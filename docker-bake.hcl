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
  targets = ["api", "proxy", "dashboard"]
}

group "go-services" {
  targets = ["api", "proxy", "loadgen"]
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
  args = {
    SERVICE = "falseflag-api"
  }
}

target "proxy-meta" {}
target "proxy" {
  inherits = ["service-base", "proxy-meta"]
  args = {
    SERVICE = "falseflag-proxy"
  }
}

target "loadgen-meta" {}
target "loadgen" {
  inherits = ["service-base", "loadgen-meta"]
  args = {
    SERVICE = "falseflag-loadgen"
  }
}

target "dashboard-meta" {}
target "dashboard" {
  context    = "."
  dockerfile = "infra/Dockerfile.dashboard"
  platforms  = ["linux/amd64", "linux/arm64"]
}
