# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.3.0"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.3.0/serv-darwin-arm64.tar.gz"
      sha256 "e571e86dab84a2298028a1a3f53a4cf2d7d02a1105ec60367d1287669599206b"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.3.0/serv-darwin-amd64.tar.gz"
      sha256 "9c30ec8b26d45b4e7913ca5e99f9a6d08696d0bde90ec98b2a259821d11ec262"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.3.0/serv-linux-arm64.tar.gz"
      sha256 "efaf02dca8e0929e9921a3027d76de1a5029e85f83211fd6acc9b3d52dc6108c"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.3.0/serv-linux-amd64.tar.gz"
      sha256 "c2b14c6199b9a7f9376c88b92d4447fc15b01cc80a9dfb25d01c596a394bba84"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
