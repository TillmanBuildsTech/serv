# typed: false
# frozen_string_literal: true

class Serv < Formula
  desc "Cross-platform Windows service / systemd / launchd process supervisor"
  homepage "https://github.com/TillmanBuildsTech/serv"
  license "MIT"
  version "0.1.2"

  on_macos do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.2/serv-darwin-arm64.tar.gz"
      sha256 "135bd9f0e40d1a50eb958741611fcec87a145bfc8a8dc399e1c9cc651f80267f"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.2/serv-darwin-amd64.tar.gz"
      sha256 "82fc4115807c8aea30b44296a48de942396944656baa2f09e83cf74deaf3e761"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.2/serv-linux-arm64.tar.gz"
      sha256 "daa1af05a4611bd77e1fb7c6b98f65da2240678fdc7968388c672ca835b8a731"
    end
    on_intel do
      url "https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.2/serv-linux-amd64.tar.gz"
      sha256 "98518bdccab09d6534ca3696dc9598b013a4d1c51d3475161ba3eee5c47a2ef6"
    end
  end

  def install
    bin.install "serv"
  end

  test do
    assert_match "serv version", shell_output("#{bin}/serv version")
  end
end
