class Worktime < Formula
  desc "macOS 上下班时间监测菜单栏工具，通过 pmset 日志识别上下班时间"
  homepage "https://github.com/Soarkey/worktime"
  url "https://github.com/Soarkey/worktime/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "3cc99b128b997eae06b64462c5193b54d04236b81f6bd7adeeabdfc8c1566e41"
  license "MIT"
  head "https://github.com/Soarkey/worktime.git", branch: "master"

  depends_on "go" => :build
  depends_on :macos

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/worktime"
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
    assert_match "上下班时间监测", shell_output("#{bin}/worktime --help")
  end
end
