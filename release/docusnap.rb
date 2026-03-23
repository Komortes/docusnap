class Docusnap < Formula
  desc "Local-first CLI for repository snapshots and generated documentation"
  homepage "https://github.com/oleksandrskoruk/docusnap"
  version "0.0.0-test"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/oleksandrskoruk/docusnap/releases/download/v0.0.0-test/docusnap-v0.0.0-test-darwin-arm64.tar.gz"
      sha256 "1fd4d5a470fda49b81d587af0922a1bd4248b467325360a182d527a855888645"
    else
      url "https://github.com/oleksandrskoruk/docusnap/releases/download/v0.0.0-test/docusnap-v0.0.0-test-darwin-amd64.tar.gz"
      sha256 "d1e3a027bde673659e90dbcd9ffbdf6d241dc28c8ad713c3b401e2bef897f36c"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/oleksandrskoruk/docusnap/releases/download/v0.0.0-test/docusnap-v0.0.0-test-linux-arm64.tar.gz"
      sha256 "749f18fce79e1714589e5de8e28bba8839bc4a1803b7b42181112addecd7f5c5"
    else
      url "https://github.com/oleksandrskoruk/docusnap/releases/download/v0.0.0-test/docusnap-v0.0.0-test-linux-amd64.tar.gz"
      sha256 "a20799d31a6e6eb8569dd96a0530219c27c090ced25c929f35f571cf5f19510c"
    end
  end

  def install
    bin.install Dir["**/docusnap"].first => "docusnap"
    pkgshare.install Dir["**/README.md"].first if Dir["**/README.md"].any?
  end

  test do
    assert_match "DocuSnap", shell_output("#{bin}/docusnap version")
  end
end
