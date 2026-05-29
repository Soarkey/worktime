class Worktime < Formula
  desc "macOS 上下班时间监测菜单栏工具，通过 pmset 日志识别上下班时间"
  homepage "https://github.com/Soarkey/worktime"
  url "https://github.com/Soarkey/worktime/releases/download/v0.1.2/worktime_0.1.2_darwin_arm64.tar.gz"
  sha256 "9b6c10827bf736200a155efe8aeebf22413879f5e211a7cd2e67dff08673c8ef"
  license "MIT"

  depends_on :macos

  def install
    bin.install "worktime"
  end

  test do
    assert_match "上下班时间监测", shell_output("#{bin}/worktime --help")
  end
end
