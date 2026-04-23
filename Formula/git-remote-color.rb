class GitRemoteColor < Formula
  desc "Colorized git remote viewer"
  homepage "https://github.com/cumulus13/git-remote-color"
  url "https://github.com/cumulus13/git-remote-color/releases/download/v1.0.6/git-remote-color-darwin-amd64"
  version "1.0.7"
  sha256 "PUT_REAL_SHA256_HERE"

  def install
    bin.install "git-remote-color-darwin-amd64" => "git-remote-color"
  end
end