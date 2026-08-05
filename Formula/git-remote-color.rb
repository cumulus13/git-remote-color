class GitRemoteColor < Formula
  desc "Colorized, GitHub-aware replacement for git remote -v with rich metadata"
  homepage "https://github.com/cumulus13/git-remote-color"
  version "1.0.17"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/cumulus13/git-remote-color/releases/download/v#{version}/git-remote-color_#{version}_darwin_amd64"
      sha256 "PUT_REAL_SHA256_DARWIN_AMD64_HERE"
    end

    on_arm do
      url "https://github.com/cumulus13/git-remote-color/releases/download/v#{version}/git-remote-color_#{version}_darwin_arm64"
      sha256 "PUT_REAL_SHA256_DARWIN_ARM64_HERE"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/cumulus13/git-remote-color/releases/download/v#{version}/git-remote-color_#{version}_linux_amd64"
      sha256 "PUT_REAL_SHA256_LINUX_AMD64_HERE"
    end

    on_arm do
      url "https://github.com/cumulus13/git-remote-color/releases/download/v#{version}/git-remote-color_#{version}_linux_arm64"
      sha256 "PUT_REAL_SHA256_LINUX_ARM64_HERE"
    end
  end

  def install
    bin.install Dir["git-remote-color_*"].first => "git-remote-color"
  end

  test do
    assert_match "git-remote-color", shell_output("#{bin}/git-remote-color --version")
  end
end
