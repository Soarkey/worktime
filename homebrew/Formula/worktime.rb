class Worktime < Formula
  desc "macOS 上下班时间监测菜单栏工具，通过 pmset 日志识别上下班时间"
  homepage "https://github.com/Soarkey/worktime"
  url "https://github.com/Soarkey/worktime/releases/download/v0.1.1/worktime_v0.1.1_darwin_arm64.tar.gz"
  sha256 "f26b6b4f82c6a3a65023f305eb0d5f5e7748bcffda24a25efe53f16e21d13476"
  license "MIT"

  depends_on :macos

  def install
    bin.install "worktime"
  end

  test do
    assert_match "上下班时间监测", shell_output("#{bin}/worktime --help")
  end
end
