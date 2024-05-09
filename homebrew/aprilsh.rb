class Aprilsh < Formula
  desc "Remote shell support intermittent or mobile network"
  homepage "https://github.com/ericwq/aprilsh"
  url "https://github.com/ericwq/aprilsh/archive/refs/tags/0.6.40.tar.gz"
  sha256 "938876efe036eb149d458c4952a989d0864ea9984a22ef704d02a902d1896826"
  license "MIT"
  # revision 1

  depends_on "go" => [:build, :test]
  depends_on "protobuf" => [:build, :test]
  uses_from_macos "ncurses", since: :monterey

  def install
    # ENV["GOPATH"] = buildpath
    ENV["GO111MODULE"] = "auto"
    go_module_ = "github.com/ericwq/aprilsh"
    git_commit = "ba85f89" # git rev-parse --short HEAD
    git_branch = "HEAD" # git rev-parse --abbrev-ref HEAD
    go_version = "go1.22.3" # go version | grep 'version' | awk '{print $3}'
    # build_time = shell_output("date -u +%Y-%m-%dT%H:%M:%S").strip
    build_time = DateTime.now.rfc3339
    output = "apsh"
    ldflags = %W[
      -s -w
      -X #{go_module_}/frontend.BuildTime=#{build_time}
      -X #{go_module_}/frontend.GitBranch=#{git_branch}
      -X #{go_module_}/frontend.GitCommit=#{git_commit}
      -X #{go_module_}/frontend.GitTag=#{version}
      -X #{go_module_}/frontend.GoVersion=#{go_version}
    ]
    system "go", "build", *std_go_args(output:, ldflags:), "./frontend/client/"
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
    # assert_match "OK", shell_output("go test ./encrypt/...")
  end
end
