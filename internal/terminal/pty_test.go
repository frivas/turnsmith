//go:build darwin || linux

package terminal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// This subprocess gives the adapter real terminal file descriptors while the
// parent checks termios restoration and reads the alternate-screen exit. Python
// supplies only the platform's PTY plumbing; no extra Go dependency is needed.
func TestTerminalPTY(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is required for the PTY regression")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, python, "-c", ptyRegression, executable)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("PTY regression: %v\n%s", err, output)
	}
	t.Log(string(output))
}

func TestTerminalPTYHelper(t *testing.T) {
	mode := os.Getenv("TURNSMITH_PTY_HELPER")
	if mode == "" {
		return
	}
	ctx := context.Background()
	if mode == "cancel" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
	}
	var states []string
	err := RunGuide(ctx, os.Stdin, os.Stdout, 3, func(index, width, height int, action string) (string, error) {
		states = append(states, fmt.Sprintf("%d:%s:%dx%d", index, action, width, height))
		if mode == "error" && index == 1 {
			return "", errors.New("sentinel")
		}
		return fmt.Sprintf("step%d\n%s", index, action), nil
	})
	fmt.Printf("RESTORED err=%v states=%s\n", err, strings.Join(states, ","))
	if mode == "reuse" {
		text, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Printf("REUSE %q %v\n", text, err)
	}
}

const ptyRegression = `
import os,pty,termios,fcntl,struct,subprocess,select,time,signal,sys
def run(mode,actions,expected):
 master,slave=pty.openpty()
 fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',24,80,0,0))
 before=termios.tcgetattr(slave)
 proc=subprocess.Popen([sys.argv[1],'-test.run=^TestTerminalPTYHelper$'],stdin=slave,stdout=slave,stderr=slave,env={**os.environ,'TERM':'xterm-256color','TCELL_ALTSCREEN':'enable','TURNSMITH_PTY_HELPER':mode})
 data=b''
 def collect(duration):
  nonlocal data
  end=time.monotonic()+duration
  while time.monotonic()<end:
   if select.select([master],[],[],max(0,end-time.monotonic()))[0]:
    try:data+=os.read(master,65536)
    except OSError:break
 try:
  collect(.15)
  for value in actions:
   if isinstance(value,tuple):
    rows,cols=value
    fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',rows,cols,0,0))
    proc.send_signal(signal.SIGWINCH)
   else:os.write(master,value)
   collect(.08)
  if mode=='reuse':
   assert b'RESTORED' in data,data[-1000:]
   os.write(master,b'canonical input\n')
  collect(.45)
  try:proc.wait(timeout=2)
  except subprocess.TimeoutExpired:raise AssertionError('session left a blocked input reader')
  collect(.05)
  after=termios.tcgetattr(slave)
  # macOS sets PENDIN when restoring canonical input; it is a transient kernel
  # retype request rather than a changed input/echo/control mode.
  after[3] &= ~getattr(termios,'PENDIN',0)
  before[3] &= ~getattr(termios,'PENDIN',0)
  assert before==after,('terminal attributes changed',before,after)
  assert proc.returncode==0,(proc.returncode,data[-2000:])
  for part in expected:assert part in data,(part,data[-2000:])
  if mode=='resize':assert b'1:next' not in data,data[-2000:]
  assert b'\x1b[?1049l' in data,('alternate screen not restored',data[-1000:])
  print('PASS',mode)
 finally:
  if proc.poll() is None:proc.kill();proc.wait()
  os.close(master);os.close(slave)
run('navigation',[b'\x1bOC',b'\x1bOD',b' ',b'r',b'\x1bOF',b'q'],[b'0:start',b'1:next',b'0:previous',b'0:reset',b'3:end'])
run('resize',[(10,20),b'n',(24,80),b'q'],[b'0:resize:80x24'])
run('interrupt',[b'\x03'],[b'context canceled'])
run('eof',[b'\x04'],[b'RESTORED err=<nil>'])
run('cancel',[],[b'context deadline exceeded'])
run('error',[b'n'],[b'sentinel'])
run('reuse',[b'q'],[b'REUSE "canonical input\\n" <nil>'])
`
