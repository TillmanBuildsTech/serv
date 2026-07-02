# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.0"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.0/serv-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_RELEASE_TARBALL"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.0/serv-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_RELEASE_TARBALL"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.0/serv-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_RELEASE_TARBALL"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.0/serv-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_OF_RELEASE_TARBALL"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
