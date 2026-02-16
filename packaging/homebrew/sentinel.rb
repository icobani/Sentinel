class Sentinel < Formula
  desc "Cross-platform file system watcher with real-time webhook notifications"
  homepage "https://github.com/icobani/Sentinel"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-darwin-arm64.tar.gz"
      sha256 "28b6d3d9d6cd4c94f8e3ce5fda916b05f49a8536b12ea3e5ab1a6b0fb5886dba"
    else
      url "https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-darwin-amd64.tar.gz"
      sha256 "f973f46404c17afe5a4bdd4d3197091fc08a943d916650f8d2e0ac270589c7fb"
    end
  end

  on_linux do
    url "https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-linux-amd64.tar.gz"
    sha256 "f51583aa7425854b104d1a81ac333b101a6a2a8892d0f244fdab1838e4110317"
  end

  def install
    bin.install "sentinel"
    etc.install "sentinel.yaml.example" => "sentinel.yaml" unless (etc/"sentinel.yaml").exist?
  end

  service do
    run [opt_bin/"sentinel"]
    working_dir etc
    keep_alive true
    log_path var/"log/sentinel.log"
    error_log_path var/"log/sentinel.log"
  end

  test do
    assert_match "sentinel", shell_output("#{bin}/sentinel version 2>&1", 0)
  end
end
