# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.3"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.3/serv-darwin-arm64.tar.gz"
      sha256 "f763e3718610633ceed3c7514f5dd1177df038b441f5b4c4dfdc6d7b6c48d593"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.3/serv-darwin-amd64.tar.gz"
      sha256 "cb5070c2c91faab2392c8daac9684e452bd96db4a5bae9a0d348324dce75a308"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.3/serv-linux-arm64.tar.gz"
      sha256 "f7496c47ab1fcceb244af6ae3399b783d8d407b0e2560e0151ba02f4504263d8"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.3/serv-linux-amd64.tar.gz"
      sha256 "05d1014258e2cf93fa64e389dbd298dd48bbb5f4af25061010b3703d297dc933"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
