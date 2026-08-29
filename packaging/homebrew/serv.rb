# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.2.0"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.2.0/serv-darwin-arm64.tar.gz"
      sha256 "bbbf168963a3b96f6ec5934f67479961410698fbd8aaa8857659a3e497a831c7"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.2.0/serv-darwin-amd64.tar.gz"
      sha256 "49c03cf886241fd3ea0ea5ab35ae10c85e19601f7ae8ead4c3959ab0504ea9f6"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.2.0/serv-linux-arm64.tar.gz"
      sha256 "490e30a15eae450258d60dbf7181a4987297db33ace95bc2aef8eb06c9acce46"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.2.0/serv-linux-amd64.tar.gz"
      sha256 "b963899587c6d68cd2309717cbab3c9904790cc132390b1b51b8b07bd6cf5a2d"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
