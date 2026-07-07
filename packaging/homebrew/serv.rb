# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.7"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.7/serv-darwin-arm64.tar.gz"
      sha256 "01146fb601f84917a64762f6cab896fff078aba2a1e66f0cde7922015fab7d46"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.7/serv-darwin-amd64.tar.gz"
      sha256 "52b9cb404e272c22ccddd98d491ab579bd21f93afe5227ab472d8c12bab85f3c"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.7/serv-linux-arm64.tar.gz"
      sha256 "79c10e6b03af7e486c37f08bc645493cbcde41a0568369807d6d1e082b219df0"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.7/serv-linux-amd64.tar.gz"
      sha256 "f528aee45eeebf1242f36d61b947e05eee8833b06fcffab844fbf3a9ae14dd0b"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
