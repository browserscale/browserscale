# Draft formula for submission to homebrew/core, which is what makes a bare
# `brew install browserscale` work without tapping first. NOT used by the
# release pipeline — that publishes a cask to our own tap instead.
#
# Eligibility: 30 forks, 30 watchers or 75 stars — tripled to 90/90/225 for a
# self-submission by the repository owner — and a repository older than 30
# days. homebrew/core also requires command-line software to be built from
# source, which is why this exists next to the generated cask rather than
# replacing it.
#
# When it lands, put the sibling tap_migrations.json in the root of the tap
# repository and delete Casks/browserscale.rb there, so tap users are moved
# over by a plain `brew update` instead of being stranded on the cask.
#
# Before submitting: set the tag in `url` and fill in `sha256` with
#   curl -sL https://github.com/browserscale/browserscale/archive/refs/tags/vX.Y.Z.tar.gz | shasum -a 256
# Afterwards `brew bump-formula-pr` keeps both up to date on every release.
class Browserscale < Formula
  desc "Command line for browserscale.cloud cloud browser automation"
  homepage "https://browserscale.cloud"
  url "https://github.com/browserscale/browserscale/archive/refs/tags/v0.3.0.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  license "MIT"
  head "https://github.com/browserscale/browserscale.git", branch: "main"

  depends_on "go" => :build

  def install
    # Homebrew builds without network access. This module has no third-party
    # dependencies, so `go build` needs nothing beyond the tarball.
    ldflags = "-s -w -X github.com/browserscale/browserscale/internal/cli.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags)
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/browserscale version")

    # Scaffolding is fully offline (templates and docs are embedded), so this
    # exercises the real work without touching the network.
    system bin/"browserscale", "init", "-name", "demo", "-yes", "-dir", testpath/"demo"
    assert_path_exists testpath/"demo/browserscale.yaml"
    assert_path_exists testpath/"demo/docs/api-reference/go.md"
  end
end
