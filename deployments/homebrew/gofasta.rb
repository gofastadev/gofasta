# Homebrew formula for gofasta
# To use this, create a tap repository: github.com/gofastadev/homebrew-tap
# Then place this file at Formula/gofasta.rb in that repo.
#
# Install: brew install gofastadev/tap/gofasta
# Update: brew upgrade gofasta

class Gofasta < Formula
  desc "A production-grade Go web framework with REST + GraphQL, DI, and full-stack features"
  homepage "https://github.com/gofastadev/gofasta"
  version "0.1.0"  # Update on each release
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/gofastadev/gofasta/releases/download/v#{version}/gofasta-darwin-arm64"
      sha256 "UPDATE_SHA256_HERE"

      def install
        bin.install "gofasta-darwin-arm64" => "gofasta"
      end
    else
      url "https://github.com/gofastadev/gofasta/releases/download/v#{version}/gofasta-darwin-amd64"
      sha256 "UPDATE_SHA256_HERE"

      def install
        bin.install "gofasta-darwin-amd64" => "gofasta"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/gofastadev/gofasta/releases/download/v#{version}/gofasta-linux-arm64"
      sha256 "UPDATE_SHA256_HERE"

      def install
        bin.install "gofasta-linux-arm64" => "gofasta"
      end
    else
      url "https://github.com/gofastadev/gofasta/releases/download/v#{version}/gofasta-linux-amd64"
      sha256 "UPDATE_SHA256_HERE"

      def install
        bin.install "gofasta-linux-amd64" => "gofasta"
      end
    end
  end

  test do
    assert_match "Gofasta", shell_output("#{bin}/gofasta --help")
  end
end
