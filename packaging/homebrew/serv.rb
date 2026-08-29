# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.9"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.9/serv-darwin-arm64.tar.gz"
      sha256 "a5ff14d6a5ea758cdbc15a0459a7160de6b04ff1afb7d63a30af0d74fccef747"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.9/serv-darwin-amd64.tar.gz"
      sha256 "9402c1302d507ff254071e2a3e0865c30c19a0a27beff6649d2a0eb7228299e6"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.9/serv-linux-arm64.tar.gz"
      sha256 "a7c5ab2606a347f918117ef9f98b4b8e740f5f549724b277a22d1c7e50d7190d"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.9/serv-linux-amd64.tar.gz"
      sha256 "f996dcdff55deb992c771a1b05d3e434854f7bb3ec06ce968f828680b7f0cef3"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
