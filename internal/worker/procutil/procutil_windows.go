//go:build windows

package procutil

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// StartWithCleanup starts the command and assigns the child process to a
// Windows Job Object with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE. When the
// worker exits (even via crash), the OS closes the job handle and kills all
// processes in the job.
func StartWithCleanup(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("create job object: %w", err)
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("set job object info: %w", err)
	}

	handle, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("open child process: %w", err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.AssignProcessToJobObject(job, handle); err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("assign process to job: %w", err)
	}

	// Intentionally do NOT close the job handle. It must stay open for
	// the lifetime of the worker so the kernel kills children on exit.
	return nil
}
