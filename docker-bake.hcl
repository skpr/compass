variable "PHP_VERSIONS" {
  default = ["8.2", "8.3", "8.4", "8.5"]
}

variable "VERSION" {
  default = "latest"
}

variable "PLATFORMS" {
  default = ["linux/amd64", "linux/arm64"]
}

# Common target: Everything inherits from this
target "_common" {
  platforms = PLATFORMS

  secret = [
    # Used to ensure we do not hit rate limiting with Mise.
    "id=github_token,env=GITHUB_TOKEN"
  ]
}

group "default" {
  targets = [
    "extension",
    "cli",
    "sidecar",
  ]
}

target "extension" {
  inherits = ["_common"]

  name = "app-${replace(PHP_VERSION, ".", "-")}"

  matrix = {
    PHP_VERSION = PHP_VERSIONS
  }

  context = "./extension"

  contexts = {
    from_image = "docker-image://ghcr.io/skpr/php-cli:${PHP_VERSION}-v2-stable"
  }

  args = {
    PHP_VERSION = PHP_VERSION
  }

  tags = [
    "ghcr.io/skpr/compass-extension:${VERSION}-${PHP_VERSION}"
  ]
}

target "cli" {
  inherits = ["_common"]

  context = "./tracing/cli"

  contexts = {
    from_image = "docker-image://docker.io/alpine:3.22"
  }

  tags = [
    "ghcr.io/skpr/compass:${VERSION}"
  ]
}

target "sidecar" {
  inherits = ["_common"]

  context = "./tracing/sidecar"

  contexts = {
    from_image = "docker-image://docker.io/alpine:3.22"
  }

  tags = [
    "ghcr.io/skpr/compass-sidecar:${VERSION}"
  ]
}
