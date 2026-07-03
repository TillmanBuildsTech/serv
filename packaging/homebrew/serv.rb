# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.6"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.6/serv-darwin-arm64.tar.gz"
      sha256 "6d27557f5f2f1b45fb5d151656e9876e26189667f577a80feb9e103a0e066299"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.6/serv-darwin-amd64.tar.gz"
      sha256 "f1b11a04b178873ff335e0a499ca548067ac352e8084d13319c7604f230f2948"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.6/serv-linux-arm64.tar.gz"
      sha256 "855b693b4ba32299998abaa9e03c5013ed7ffbe2330ee4370384fff49758a1b8"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.6/serv-linux-amd64.tar.gz"
      sha256 "adf8d6bd8640caa401bdb97b530f0615e2728249df7787a3e11c84dcb59aee9c"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
