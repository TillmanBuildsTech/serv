# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.5"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.5/serv-darwin-arm64.tar.gz"
      sha256 "34842587de5faf034a22e850d64cddcca0b51df5f3e3527bb3f7ef1128cb3a0e"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.5/serv-darwin-amd64.tar.gz"
      sha256 "1cf38c52be3a288fd5cb4d9f98524d2d848b2c446fe475202722ea8514d6b92f"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.5/serv-linux-arm64.tar.gz"
      sha256 "73c070221877fecaeb7f8f807c0d59b797fb20614d7636ff3d82ec65f07ab13e"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.5/serv-linux-amd64.tar.gz"
      sha256 "b31e316731017f5acd330da26d2a18fdd073decedfe5504ef8be93e9bef4e406"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
