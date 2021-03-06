package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

/*
golang 写一个 agent, 主要实现这么几个功能
1. 这个agent 的作用是启动/停止 各种二进制程序和查看二进制程序的状态
2.这里会涉及到信号的处理(比如外部可以通过调用 agent 的 api 进行对二进制程序的 kill/down 等操作)
3. 有的程序一次会起不来，可能需要retry
4. 程序可能还会接收其他各种信号
 */
//根据进程名字查询出进程id
func FindProcess(serverName string)(bool,int){
	if runtime.GOOS=="linux"{
		cmdStr:=fmt.Sprintf("ps -ef|grep %s|grep -v grep|awk '{print $2}'",serverName)
		cmd:=exec.Command("/bin/sh", "-c", cmdStr)
		if out,err:=cmd.Output();err!=nil{
			return false,0
		}else if x:=strings.Fields(string(out));len(x)<=0{
			return false,0
		}else if pid,err:=strconv.Atoi(x[0]);err!=nil{
			return false,0
		}else{
			return true,pid
		}
		return false,0
	}
	x:=strings.Fields(StatusProcess(serverName))
	if len(x)==0{
		return false,0
	}
	for i:=1;i<len(x);i++{
		if pid,err:=strconv.Atoi(x[i]);err==nil&&x[i-1]==serverName{
			return true,pid
		}
	}
	return false,0
}
//查询二进制进程状态
func StatusProcess(serverName string)string{
	var cmdStr string
	var cmd *exec.Cmd
	if runtime.GOOS=="linux"{
		cmdStr=fmt.Sprintf("ps -ef|grep %s|grep -v grep",serverName)
		cmd=exec.Command("/bin/sh", "-c", cmdStr)
	}else{
		cmdStr = fmt.Sprintf("tasklist | findstr %s",serverName)
		cmd=exec.Command("cmd","/C",cmdStr)
	}
	out,err:=cmd.Output()
	if err!=nil{
		return err.Error()
	}
	return string(out)
}
//关闭进程
func KillProcess(serverName string){
	if ok,pid:=FindProcess(serverName);ok==false{
		fmt.Println("not found")
	}else if pro,err:=os.FindProcess(pid);err!=nil{
		fmt.Println(err)
	}else if err:=pro.Kill();err!=nil{
		fmt.Println("kill fail",serverName)
	}else{
		fmt.Println("kill successful",serverName)
	}
}
//开启进程
func StartProcess(serverName string) error {
	fmt.Println(serverName,"start")
	var cmd *exec.Cmd
	if runtime.GOOS=="windows"{
		cmd=exec.Command("cmd.exe", "/C", "start", serverName)
	}else{
		cmd=exec.Command("/bin/sh", "-c", "nohup",serverName)
	}
	//cmd := exec.Command(serverName)
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		fmt.Println(serverName, "start failed")
		return err
	}
	// 正常日志
	logScan := bufio.NewScanner(stdout)
	go func() {
		for logScan.Scan() {
			fmt.Println(logScan.Text())
		}
	}()
	// 错误日志
	errBuf := bytes.NewBufferString("")
	scan := bufio.NewScanner(stderr)
	cmd.Wait()
	if !cmd.ProcessState.Success() {
		// 执行失败，返回错误信息
		for scan.Scan() {
				s := scan.Text()
				fmt.Println("error: ", s)
				errBuf.WriteString(s)
				errBuf.WriteString("\n")
			}
		return errors.New(errBuf.String())
	}
	fmt.Println("finished")
	return nil
}
//信号处理
func SignalProcess(serverName string,signal string){
	if ok,pid:=FindProcess(serverName);ok==false {
		fmt.Println("not found")
	}else if pro,err:=os.FindProcess(pid);err!=nil {
		fmt.Println(err)
	}else{
		switch signal {
		case "sigquit":
			pro.Signal(syscall.SIGQUIT)
		case "sigkill":
			pro.Signal(syscall.SIGKILL)
		case "sigint":
			pro.Signal(syscall.SIGINT)
		default:
			fmt.Println("other signal")
		}
	}
	fmt.Println(serverName)

}
var fileName string
var cmd string
var signal string
func main() {
	flag.StringVar(&fileName, "file", "", "程序名称")
	flag.StringVar(&cmd, "cmd", "", "指令")
	flag.StringVar(&cmd, "signal", "", "信号")
	flag.Parse()
	if cmd == "start" {
		for {
			if err:=StartProcess(fileName);err==nil{
				break
			}else{
				fmt.Println(err)
				fmt.Println(fileName,"restart")
				time.Sleep(time.Second)
			}
		}
	} else if cmd == "kill" {
		KillProcess(fileName)
	} else if cmd == "status" {
		fmt.Println(StatusProcess(fileName))
	} else if len(signal) > 0 {
		SignalProcess(fileName, signal)
	} else {
		fmt.Println("未知命令")
	}
}