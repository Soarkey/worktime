package menubar

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Soarkey/worktime/internal/attendance"
)

func exportCSV() {
	path, err := chooseSavePath()
	if err != nil || path == "" {
		return
	}

	records, err := attendance.GetAll()
	if err != nil || len(records) == 0 {
		return
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].WorkDate < records[j].WorkDate
	})

	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(f)
	w.Write([]string{"日期", "上班时间", "预计下班", "实际下班", "迟到(分钟)"})
	for _, r := range records {
		w.Write([]string{r.WorkDate, r.StartTime, r.ExpectedLeave, r.ActualLeave, fmt.Sprintf("%d", r.LateMinutes)})
	}
	w.Flush()

	exec.Command("osascript", "-e", fmt.Sprintf(`display notification "已导出 %d 条记录" with title "worktime"`, len(records))).Run()
}

func chooseSavePath() (string, error) {
	script := `set f to choose file name with prompt "导出考勤记录" default name "worktime.csv"
return POSIX path of f`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
