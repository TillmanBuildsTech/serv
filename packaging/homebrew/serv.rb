# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.4"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.4/serv-darwin-arm64.tar.gz"
      sha256 "9e5e731161c1c6a3f8aa487f7d86766b1076fd23c4c004b7228badfea17ccd79"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.4/serv-darwin-amd64.tar.gz"
      sha256 "59fca61f0d3cd1b72421a04f5e4e047e91e6622d822ace7c6e5188d6a5dc5148"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.4/serv-linux-arm64.tar.gz"
      sha256 "1da25549d6c46d6e7bcc09ab83266594fd8d4524921468ec5ceb9ddd18c029a6"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.4/serv-linux-amd64.tar.gz"
      sha256 "5ea4708c46512e3e39ce0c10e7ee73a076ad59986280912946587be7169a21cd"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
