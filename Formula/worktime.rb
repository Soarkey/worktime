class Worktime < Formula
  desc "macOS 自动考勤菜单栏工具，通过 pmset 日志识别上下班时间"
  homepage "https://github.com/Soarkey/worktime"
  url "https://github.com/Soarkey/worktime/archive/refs/tags/v0.1.0.tar.gz"
  sha256 ""
  license "MIT"

  depends_on :macos

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/worktime"
  end

  def caveats
    <<~EOS
      启动后台服务:
        worktime daemon

      安装开机自启:
        worktime install

      查看状态:
        worktime status
    EOS
  end

  test do
    assert_match "macOS 自动考勤", shell_output("#{bin}/worktime --help")
  end
end
