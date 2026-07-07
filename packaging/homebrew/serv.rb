# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.8"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.8/serv-darwin-arm64.tar.gz"
      sha256 "ddb1e45b413828beb8a125d229bb86e669b34c09031dc586caf63ce1e88fe2ac"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.8/serv-darwin-amd64.tar.gz"
      sha256 "4c76a7574ff4a2c0556e4acebe76e6462e7bce679d2a89380e152c7d57ba3f2e"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.8/serv-linux-arm64.tar.gz"
      sha256 "702d707326dd397bbc73450de835e39382a25c5115bce78f6794a98c798c2cd5"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.8/serv-linux-amd64.tar.gz"
      sha256 "ffbc62a92ef6e291dc011f7fd47f9f75ed03617612727cc7ca2ed12d77c2826d"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
