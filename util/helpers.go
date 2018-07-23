package util

import (
  "runtime"
  "os/exec"
	"strings"
	"fmt"
)

func WarnIfGitAutoCrlfEnabled() {
  if runtime.GOOS == "windows" {
    cmd := exec.Command("git", "config", "core.autocrlf")
    stdout, err := cmd.Output()
    if err == nil {
      autocrlf := strings.TrimSpace(string(stdout))
      if autocrlf == "true" {
        fmt.Println(`WARNING: Git core.autcrlf is enabled
This option may cause unexpected errors in Bash scripts.
It is recommended that you disable this feature by running:

    C:\> git config --global core.autocrlf false

Then rewrite the Git index to pick up all the new line endings
by running 'git reset' on your repo.`)
      }
    }
  }
}
