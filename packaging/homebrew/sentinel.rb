class Sentinel < Formula
  desc "Cross-platform file system watcher with real-time webhook notifications"
  homepage "https://github.com/icobani/Sentinel"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-darwin-arm64.tar.gz"
      sha256 "96fef64a730ab6b820a9982dd048b1db2368ddbe64e23fed8589f3d9b88caca7"
    else
      url "https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-darwin-amd64.tar.gz"
      sha256 "9033f8a00466e62cf290c2aa7cbb57a846f479db2a59ffb954a32162533cf996"
    end
  end

  on_linux do
    url "https://github.com/icobani/Sentinel/releases/download/v1.0.0/sentinel-linux-amd64.tar.gz"
    sha256 "4d4573a0a11a927e32f9481751338072a5c32ac6ec48ec7d945faa10c61dfac9"
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
