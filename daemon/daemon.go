package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// the special environment variable for this sub process to background running
const ENV_NAME = "BK_DAEMON_IDX"

// call the function of background's count, used to judge the main or sub process.
var runIdx int = 0

// daemon process
type Daemon struct {
	LogFile     string //record the std and error output of daemon process or sub process. do not record if logFile is empty.
	MaxCount    int    //the max count for restarting the sub process. Tt is infinite if it set to zero.
	MaxError    int    //the max count of sub process exiting when some error or exception occured. when over this count, the sub process will not restart.
	MinExitTime int64  //th min time duration of running of sub process. over this time, the sub process will be as an exception.
}

// Background, start a sub process background, and exit itself.
// logFile, if it is not empty, the stderr and stdout will output it.
// isExit, if true, the main process exit(0), else return *os.Process in the main process and nil in the sub process.
func Background(logFile string, isExit bool) (*exec.Cmd, error) {
	runIdx++
	envIdx, err := strconv.Atoi(os.Getenv(ENV_NAME))
	if err != nil {
		envIdx = 0
	}
	if runIdx <= envIdx { //if sub process call this function, directly return.
		return nil, nil
	}

	//set the evironment available of sub process
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%d", ENV_NAME, runIdx))

	//start sub process
	cmd, err := startProc(os.Args, env, logFile)
	if err != nil {
		fmt.Println(os.Getpid(), "process start error : ", err)
		return nil, err
	} else {
		fmt.Println("process start success!")
	}

	if isExit {
		os.Exit(0)
	}
	return cmd, nil
}

// wait for a kill(9) single to exit
func WaitForKill() {
	if pid := os.Getpid(); pid != 1 {
		os.WriteFile("process.pid", []byte(strconv.Itoa(pid)), 0666)
		os.WriteFile("stop.sh", []byte("kill `cat process.pid`"), 0777)
		defer os.Remove("process.pid")
		defer os.Remove("stop.sh")
	}
	ch := make(chan os.Signal, 1)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	fmt.Printf("process stop. %d : %d \n", s, os.Getpid())
}

func NewDaemon(logFile string) *Daemon {
	return &Daemon{
		LogFile:     logFile,
		MaxCount:    0,
		MaxError:    3,
		MinExitTime: 10,
	}
}

// run the daemon process
func (d *Daemon) Run() {
	Background(d.LogFile, true)

	//daemon process start a sub process, and watch it in a loop
	var t int64
	count := 1
	errNum := 0
	for {
		//daemon information
		dInfo := fmt.Sprintf("daemon process(pid:%d; count:%d/%d; errNum:%d/%d):",
			os.Getpid(), count, d.MaxCount, errNum, d.MaxError)
		if errNum > d.MaxError {
			fmt.Println(dInfo, "too many count of failing to start sub process, it will exit(1)")
			os.Exit(1)
		}
		if d.MaxCount > 0 && count > d.MaxCount {
			fmt.Println(dInfo, "too many count of restart the sub process, it will exit(0)")
			os.Exit(0)
		}
		count++

		t = time.Now().Unix()
		cmd, err := Background(d.LogFile, false)
		if err != nil {
			fmt.Println(dInfo, "sub process start failed. ", "err : ", err)
			errNum++
			continue
		}

		if cmd == nil {
			fmt.Printf("sub process pid = %d : start running ...", os.Getpid())
			break
		}

		//main process wating for the sub process to exit
		err = cmd.Wait()
		dat := time.Now().Unix() - t //the time of sub process running
		if dat < d.MinExitTime {     //exit for exception
			errNum++
		} else {
			errNum = 0
		}
		fmt.Printf("%s watch process(%d) exit, total time of run is %d seconds : %v\n", dInfo, cmd.ProcessState.Pid(), dat, err)
	}
}

// start a new process
func startProc(args, env []string, logFile string) (*exec.Cmd, error) {
	cmd := &exec.Cmd{
		Path: args[0],
		Args: args,
		Env:  env,
		SysProcAttr: &syscall.SysProcAttr{
			Setsid: true,
		},
	}

	if logFile != "" {
		stdout, err := os.OpenFile(logFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(os.Getpid(), " : open log file error : ", err)
			return nil, err
		}
		cmd.Stderr = stdout
		cmd.Stdout = stdout
	}

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
