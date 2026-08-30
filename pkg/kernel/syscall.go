package kernel

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// Standard POSIX / Linux-like System Call Numbers
const (
	SysFork      = 1
	SysExec      = 2
	SysWait      = 3
	SysExit      = 4
	SysRead      = 5
	SysWrite     = 6
	SysPipe      = 7
	SysShmGet    = 8
	SysShmAttach = 9
	SysKill      = 10
	SysSbrk      = 11
	SysYield     = 12
	SysGetPID    = 13
)

func toErrno(err int) uint64 {
	return uint64(int64(-err))
}

var (
	ErrInvalidSyscall   = errors.New("invalid syscall: unknown syscall number")
	ErrBadUserPointer   = errors.New("bad address (EFAULT): user pointer points to protected kernel address space")
	ErrInvalidArgument  = errors.New("invalid argument (EINVAL)")
	ErrPermissionDenied = errors.New("permission denied (EACCES)")
)

// SyscallRegisters holds simulated hardware registers used in x86-64 syscall calling convention
type SyscallRegisters struct {
	RAX uint64 // Syscall Number (in) / Return value or -errno (out)
	RDI uint64 // Argument 1
	RSI uint64 // Argument 2
	RDX uint64 // Argument 3
	R10 uint64 // Argument 4
	R8  uint64 // Argument 5
	R9  uint64 // Argument 6
}

// SyscallHandlerFunc defines the signature for a kernel syscall implementation
type SyscallHandlerFunc func(pid int, regs *SyscallRegisters) (int64, error)

// SyscallEntry describes an entry in the kernel's system call dispatch table
type SyscallEntry struct {
	Number      int
	Name        string
	Signature   string
	Description string
	Handler     SyscallHandlerFunc
	TotalCalls  int
}

// SyscallDispatcher manages kernel system call routing, argument validation, and execution
type SyscallDispatcher struct {
	Table       map[int]*SyscallEntry
	KernelBase  uint64 // 0xC0000000 (Protected kernel address boundary)
	CallHistory []string
}

func NewSyscallDispatcher() *SyscallDispatcher {
	sd := &SyscallDispatcher{
		Table:       make(map[int]*SyscallEntry),
		KernelBase:  0xC0000000,
		CallHistory: make([]string, 0),
	}

	// Register Core System Calls
	sd.Register(SysGetPID, "getpid", "pid_t sys_getpid(void)", "Returns the process ID of the calling process", defaultGetPID)
	sd.Register(SysYield, "sched_yield", "int sys_yield(void)", "Voluntarily relinquishes the CPU to other ready processes", defaultYield)
	sd.Register(SysWrite, "write", "ssize_t sys_write(int fd, const void *buf, size_t count)", "Writes data from user buffer to file/socket/pipe", defaultWrite)
	sd.Register(SysRead, "read", "ssize_t sys_read(int fd, void *buf, size_t count)", "Reads data from file/socket/pipe into user buffer", defaultRead)
	sd.Register(SysSbrk, "sbrk", "void *sys_sbrk(intptr_t increment)", "Adjusts heap break pointer to grow/shrink dynamic memory", defaultSbrk)
	sd.Register(SysExit, "exit", "void sys_exit(int status)", "Terminates the calling process with exit status code", defaultExit)
	sd.Register(SysFork, "fork", "pid_t sys_fork(void)", "Duplicates calling process (clones address space & PCB)", defaultFork)
	sd.Register(SysExec, "execve", "int sys_execve(const char *filename, char *const argv[])", "Replaces current process image with new program", defaultExec)
	sd.Register(SysWait, "waitpid", "pid_t sys_waitpid(pid_t pid, int *status, int options)", "Blocks parent until specified child process exits", defaultWait)
	sd.Register(SysPipe, "pipe", "int sys_pipe(int pipefd[2])", "Creates unidirectional IPC pipe data channel", defaultPipe)
	sd.Register(SysShmGet, "shmget", "int sys_shmget(key_t key, size_t size, int shmflg)", "Allocates shared memory segment across processes", defaultShmGet)
	sd.Register(SysShmAttach, "shmat", "void *sys_shmat(int shmid, const void *shmaddr, int shmflg)", "Maps shared physical frame into virtual address space", defaultShmAttach)

	return sd
}

func (sd *SyscallDispatcher) Register(num int, name, sig, desc string, handler SyscallHandlerFunc) {
	sd.Table[num] = &SyscallEntry{
		Number:      num,
		Name:        name,
		Signature:   sig,
		Description: desc,
		Handler:     handler,
	}
}

// ValidateUserPointer ensures user-provided pointers do not point into kernel space (prevents kernel corruption)
func (sd *SyscallDispatcher) ValidateUserPointer(vAddr uint64, size uint64) error {
	if vAddr+size > sd.KernelBase || vAddr >= sd.KernelBase {
		return fmt.Errorf("%w: pointer 0x%08X (size %d) touches kernel space (>= 0x%08X)", ErrBadUserPointer, vAddr, size, sd.KernelBase)
	}
	return nil
}

// Dispatch executes the requested system call with calling convention register mechanics
func (sd *SyscallDispatcher) Dispatch(pid int, regs *SyscallRegisters) (int64, error) {
	syscallNum := int(regs.RAX)
	entry, exists := sd.Table[syscallNum]
	if !exists {
		regs.RAX = toErrno(1) // -ENOSYS
		return -1, fmt.Errorf("%w: Syscall #%d", ErrInvalidSyscall, syscallNum)
	}

	entry.TotalCalls++

	// Execute handler in Kernel Mode
	retVal, err := entry.Handler(pid, regs)
	if err != nil {
		regs.RAX = toErrno(22) // -EINVAL
		logEntry := fmt.Sprintf("[PID %d] syscall %s(%d, 0x%X, 0x%X) ──► ERROR: %v",
			pid, entry.Name, regs.RDI, regs.RSI, regs.RDX, err)
		sd.CallHistory = append(sd.CallHistory, logEntry)
		return retVal, err
	}

	regs.RAX = uint64(retVal)
	logEntry := fmt.Sprintf("[PID %d] syscall %s(arg1:0x%X, arg2:0x%X, arg3:0x%X) ──► return: %d (%s)",
		pid, entry.Name, regs.RDI, regs.RSI, regs.RDX, retVal, visualizer.Green("OK"))
	sd.CallHistory = append(sd.CallHistory, logEntry)

	return retVal, nil
}

// RenderSyscallTable visualizes the kernel syscall dispatch table
func (sd *SyscallDispatcher) RenderSyscallTable() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Kernel System Call Dispatch Table (Syscall Table)"))

	tbl := visualizer.NewTable("Syscall # (RAX)", "Name", "C Function Signature", "Description", "Calls")
	tbl.SetAlignment("center", "left", "left", "left", "right")

	for num := 1; num <= 13; num++ {
		if entry, exists := sd.Table[num]; exists {
			tbl.AddRow(
				fmt.Sprintf("%d", entry.Number),
				visualizer.Bold+entry.Name+visualizer.Reset,
				entry.Signature,
				entry.Description,
				fmt.Sprintf("%d", entry.TotalCalls),
			)
		}
	}

	sb.WriteString(tbl.Render())

	callingConv := []string{
		"SYSTEM CALL CALLING CONVENTION (x86-64 Linux ABI):",
		"• RAX = System Call Number (e.g. 1 for fork, 6 for write)",
		"• RDI = 1st Argument (e.g. file descriptor int fd)",
		"• RSI = 2nd Argument (e.g. user buffer address const void *buf)",
		"• RDX = 3rd Argument (e.g. byte count size_t count)",
		"• R10, R8, R9 = 4th, 5th, 6th Arguments",
		"• Return Value: Placed in RAX upon return (negative integer = -errno error code).",
		"",
		"KERNEL SECURITY (POINTER SANITIZATION):",
		"• The kernel MUST validate all user pointers before dereferencing them.",
		"• If a user passes an address >= 0xC0000000, kernel aborts with -EFAULT to prevent user from reading/writing kernel memory!",
	}
	sb.WriteString("\n" + visualizer.Box("Syscall Mechanism & Security", callingConv))

	return sb.String()
}

// Default Syscall Handlers
func defaultGetPID(pid int, regs *SyscallRegisters) (int64, error) {
	return int64(pid), nil
}

func defaultYield(pid int, regs *SyscallRegisters) (int64, error) {
	return 0, nil
}

func defaultWrite(pid int, regs *SyscallRegisters) (int64, error) {
	fd := regs.RDI
	bufPtr := regs.RSI
	count := regs.RDX

	// Validate pointer does not touch kernel space
	if bufPtr >= 0xC0000000 {
		return -14, fmt.Errorf("%w: buffer pointer 0x%X in kernel space", ErrBadUserPointer, bufPtr)
	}

	_ = fd
	return int64(count), nil // Pretend count bytes written
}

func defaultRead(pid int, regs *SyscallRegisters) (int64, error) {
	bufPtr := regs.RSI
	count := regs.RDX
	if bufPtr >= 0xC0000000 {
		return -14, fmt.Errorf("%w: buffer pointer 0x%X in kernel space", ErrBadUserPointer, bufPtr)
	}
	return int64(count), nil
}

func defaultSbrk(pid int, regs *SyscallRegisters) (int64, error) {
	return 0x00020000, nil // Returns old heap break
}

func defaultExit(pid int, regs *SyscallRegisters) (int64, error) {
	return 0, nil
}

func defaultFork(pid int, regs *SyscallRegisters) (int64, error) {
	return 1001, nil // Child PID in parent
}

func defaultExec(pid int, regs *SyscallRegisters) (int64, error) {
	return 0, nil
}

func defaultWait(pid int, regs *SyscallRegisters) (int64, error) {
	return 1001, nil // Reaped child PID
}

func defaultPipe(pid int, regs *SyscallRegisters) (int64, error) {
	return 0, nil
}

func defaultShmGet(pid int, regs *SyscallRegisters) (int64, error) {
	return 42, nil // Shared memory ID
}

func defaultShmAttach(pid int, regs *SyscallRegisters) (int64, error) {
	return 0x00300000, nil // Attached virtual address
}
