package extract

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunCommandReadsStdout(t *testing.T) {
	bin, args := "sh", []string{"-c", "echo parser-ok"}
	if runtime.GOOS == "windows" {
		bin, args = "cmd", []string{"/c", "echo parser-ok"}
	}
	res, err := RunCommand(context.Background(), "tika", bin, args, []byte("entrada"), time.Second)
	if err != nil || !strings.Contains(res.Text, "parser-ok") || res.Parser != "tika" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}
