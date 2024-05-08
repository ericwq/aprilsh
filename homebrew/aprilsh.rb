class Aprilsh < Formula
  desc "remote shell support intermittent or mobile network."
  homepage "https://github.com/ericwq/aprilsh"
  url "https://github.com/ericwq/aprilsh/archive/refs/tags/0.6.40.tar.gz"
  sha256 "938876efe036eb149d458c4952a989d0864ea9984a22ef704d02a902d1896826"
  license "MIT"
  revision 1

  depends_on "go" => [:build, :test]

  depends_on "protobuf"

  uses_from_macos "ncurses"

  def install
    ENV["GOPATH"] = buildpath
    ENV["GO111MODULE"] = "auto"
    _go_module = "github.com/ericwq/aprilsh"
    _git_commit = "ba85f89"   # git rev-parse --short HEAD
    _git_branch = "HEAD"	  # git rev-parse --abbrev-ref HEAD
    _go_version = "go1.21.5"  # go version | grep 'version' | awk '{print $3}'
    # _build_time = shell_output("date -u +%Y-%m-%dT%H:%M:%S").strip
    _build_time = DateTime.now().rfc3339(0)
    ldflags = %W[
        -s -w
        -X #{_go_module}/frontend.BuildTime=#{_build_time}
		-X #{_go_module}/frontend.GitBranch=#{_git_branch}
		-X #{_go_module}/frontend.GitCommit=#{_git_commit}
		-X #{_go_module}/frontend.GitTag=#{version}
		-X #{_go_module}/frontend.GoVersion=#{_go_version}
    ]
    system "go", "build", *std_go_args(ldflags:),"-o","apsh","./frontend/client/"
    bin.install "apsh"
  end

  test do
    # `test do` will create, run in and delete a temporary directory.
    #
    # This test will fail and we won't accept that! For Homebrew/homebrew-core
    # this will need to be a test that verifies the functionality of the
    # software. Run the test with `brew test aprilsh`. Options passed
    # to `brew install` such as `--HEAD` also need to be provided to `brew test`.
    #
    # The installed folder is not in the path, so use the entire path to any
    # executables being tested: `system "#{bin}/program", "do", "something"`.
    system "false"
  end
end
