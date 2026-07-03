# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.1"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.1/serv-darwin-arm64.tar.gz"
      sha256 "7aae11609874558ab80f191d2de1d23685139f8f7a0afb9ba2f559671d5745c1"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.1/serv-darwin-amd64.tar.gz"
      sha256 "fb56a1d2c482b2af44a8e8f52b8cc0c7ab229b62b7ea3380a8a5f1584a6a6c11"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.1/serv-linux-arm64.tar.gz"
      sha256 "27901cd4f7fdef0ce74d42685868cd80c672f86c19c4e38c44b876e3f6383948"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.1/serv-linux-amd64.tar.gz"
      sha256 "bd3d471c1aebb38af10ae69d2f552d497227bd7e260bbdf9e809f47f74e7937e"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
