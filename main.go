package main

import (
	// "flag"
	"os"
	"os/exec"
	"runtime"
	// "runtime/pprof"
	"sync"
	"syscall"
)

// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var (
	quit     chan struct{}
	relaunch bool
)

// This code is from goagain
func lookPath() (argv0 string, err error) {
	argv0, err = exec.LookPath(os.Args[0])
	if nil != err {
		return
	}
	if _, err = os.Stat(argv0); nil != err {
		return
	}
	return
}

func main() {
	// quit是退出信号通道
	quit = make(chan struct{})
	// Parse flags after load config to allow override options in config
	// 解析flags来覆盖默认设置
	cmdLineConfig := parseCmdLineConfig()
	if cmdLineConfig.PrintVer {
		//如果需要打印版本号
		//则打印后退出
		printVersion()
		os.Exit(0)
	}

	parseConfig(cmdLineConfig.RcFile, cmdLineConfig)

	initSelfListenAddr() // 初始化监听地址
	initLog()            // 初始化日志
	initAuth()           // 初始化用户验证
	initSiteStat()       // 初始化站点状态
	initPAC()            // 初始化PAC, initPAC uses siteStat, so must init after site stat

	initStat()

	initParentPool()

	/*
		if *cpuprofile != "" {
			f, err := os.Create(*cpuprofile)
			if err != nil {
				Fatal(err)
			}
			pprof.StartCPUProfile(f)
		}
	*/

	if config.Core > 0 {
		runtime.GOMAXPROCS(config.Core)
	}

	go sigHandler() //启动信号处理协程
	go runSSH()     //启动SSH
	if config.EstimateTimeout {
		//启动超时预计
		go runEstimateTimeout()
	} else {
		info.Println("timeout estimation disabled")
	}

	var wg sync.WaitGroup
	wg.Add(len(listenProxy))
	for _, proxy := range listenProxy {
		//启动各代理
		go proxy.Serve(&wg, quit)
	}

	wg.Wait() //等待各代理结束运行

	if relaunch { //如果程序被设置为重启
		info.Println("Relunching cow...")
		// Need to fork me.
		argv0, err := lookPath()
		if nil != err {
			errl.Println(err)
			return
		}

		err = syscall.Exec(argv0, os.Args, os.Environ()) //在当前进程中调用exec系统调用
		if err != nil {
			errl.Println(err)
		}
	}
	debug.Println("the main process is , exiting...")
}
