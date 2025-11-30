package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [process-name]",
	Short: "Stop a running project",
	Long: `Stop a running project by name or process.

Examples:
  glive stop node           # Stop Node.js processes
  glive stop python         # Stop Python processes
  glive stop go             # Stop Go processes
  glive stop 12345          # Stop process by PID`,
	Args: cobra.MaximumNArgs(1),
	Run:  stopProject,
}

var (
	stopForce bool
	stopAll   bool
)

func init() {
	stopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Force kill the process")
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop all matching processes")
}

func stopProject(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		// Try to detect and list running dev processes
		fmt.Println("Running development processes:")
		listDevProcesses()
		fmt.Println("\nUsage: glive stop <process-name|pid>")
		return
	}

	target := args[0]

	// Check if target is a PID
	if pid, err := strconv.Atoi(target); err == nil {
		stopByPID(pid)
		return
	}

	// Stop by process name
	stopByName(target)
}

func listDevProcesses() {
	// Common development process names
	devProcesses := []string{"node", "npm", "yarn", "pnpm", "python", "python3", "go", "cargo", "java", "gradle", "mvn"}

	found := false
	for _, proc := range devProcesses {
		pids := findProcesses(proc)
		if len(pids) > 0 {
			found = true
			for _, pid := range pids {
				fmt.Printf("  %s (PID: %d)\n", proc, pid)
			}
		}
	}

	if !found {
		fmt.Println("  No development processes found")
	}
}

func findProcesses(name string) []int {
	var pids []int

	if runtime.GOOS == "windows" {
		// Windows: use tasklist
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s*", name), "/FO", "CSV", "/NH").Output()
		if err != nil {
			return pids
		}

		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(name)) {
				parts := strings.Split(line, ",")
				if len(parts) >= 2 {
					pidStr := strings.Trim(parts[1], `"`)
					if pid, err := strconv.Atoi(pidStr); err == nil {
						pids = append(pids, pid)
					}
				}
			}
		}
	} else {
		// Unix: use pgrep
		out, err := exec.Command("pgrep", "-f", name).Output()
		if err != nil {
			return pids
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if pid, err := strconv.Atoi(line); err == nil {
				pids = append(pids, pid)
			}
		}
	}

	return pids
}

func stopByPID(pid int) {
	var cmd *exec.Cmd

	if stopForce {
		if runtime.GOOS == "windows" {
			cmd = exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
		} else {
			cmd = exec.Command("kill", "-9", strconv.Itoa(pid))
		}
	} else {
		if runtime.GOOS == "windows" {
			cmd = exec.Command("taskkill", "/PID", strconv.Itoa(pid))
		} else {
			cmd = exec.Command("kill", strconv.Itoa(pid))
		}
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to stop process %d: %v\n", pid, err)
		os.Exit(1)
	}

	fmt.Printf("Stopped process %d\n", pid)
}

func stopByName(name string) {
	pids := findProcesses(name)

	if len(pids) == 0 {
		fmt.Printf("No processes found matching '%s'\n", name)
		return
	}

	if len(pids) > 1 && !stopAll {
		fmt.Printf("Found %d processes matching '%s':\n", len(pids), name)
		for _, pid := range pids {
			fmt.Printf("  PID: %d\n", pid)
		}
		fmt.Println("\nUse --all to stop all, or specify a PID directly")
		return
	}

	for _, pid := range pids {
		stopByPID(pid)
	}
}
