package main
 
import (
    "os"
    "os/exec"
    "io"
    "fmt"
    "net"
    "log" 
    "time" 
    "bufio"
    "regexp"
    "strconv"
    "syscall" 
    "strings" 
    

    cpu       "github.com/shirou/gopsutil/v3/cpu"
    disk      "github.com/shirou/gopsutil/v3/disk"
    host      "github.com/shirou/gopsutil/v3/host"
    mem       "github.com/shirou/gopsutil/v3/mem"
    load      "github.com/shirou/gopsutil/v3/load"
    process "github.com/shirou/gopsutil/v3/process"
    envconfig "github.com/kelseyhightower/envconfig"
    color     "github.com/fatih/color"
    redis     "github.com/garyburd/redigo/redis"

)

type osEnv struct {
    Lang        string 
    Debug       bool
}

var HOST_IP string
 
func main() {
    HOST_IP = get_ip()
    v, _ := mem.VirtualMemory()
    c, _ := cpu.Info()
    a, _ := load.Avg()
    d, _ := disk.Usage("/")
    filesystems, _ := disk.Partitions(true)
    n, _ := host.Info()
 
    fmt.Printf("        Mem       : %v MB  Free: %v MB Usage:%f%%\n", v.Total/1024/1024 ,v.Free/1024/1024, v.UsedPercent)
    if len(c) > 1 {
        for _, sub_cpu := range c {
            modelname := sub_cpu.ModelName
            cores := sub_cpu.Cores
            fmt.Printf("        CPU       : %v   %v cores \n", modelname, cores)
        }
    } else {
        sub_cpu := c[0]
        modelname := sub_cpu.ModelName
        cores := sub_cpu.Cores
        fmt.Printf("        CPU       : %v   %v cores \n", modelname, cores)
 
    }
    fmt.Printf("        HD        : %v GB  Free: %v GB Usage:%f%%\n", d.Total/1024/1024/1024, d.Free/1024/1024/1024, d.UsedPercent)
    fmt.Printf("        OS        : %v   %v  \n", n.OS, n.PlatformVersion)
    fmt.Printf("        Hostname  : %v  \n", n.Hostname)
    fmt.Printf("        load      : %v  %v   %v\n", a.Load1,a.Load5,a.Load15)
    for  i:=0;i<len( filesystems); i++ {
       if filesystems[i].Device[0:1] == "/" {
           fmt.Printf("%v \n", filesystems[i].Mountpoint)
           ds, _ := disk.Usage( filesystems[i].Mountpoint)
           fmt.Printf("        File Systems : %v  Total: %vG   Used: %vG  Free: %vG UsedPercent: %2.2f%%, inodePercent: %2.2f%% \n", ds.Path,ds.Total/1024/1024/1024, ds.Used/1024/1024/1024, ds.Free/1024/1024/1024 , ds.UsedPercent, ds.InodesUsedPercent )
       }
    }
    sd, _ := mem.SwapDevices()
    fmt.Printf("        SwapInfo   : %v   \n", sd)
    sms, _ := mem.SwapMemory()
    fmt.Printf("        SwapMemory  : %v   \n", sms)
    var envs osEnv
    err := envconfig.Process("", &envs)
    if err != nil {
        log.Fatal(err.Error())
    }
    fmt.Printf("        EnvVar: %v   \n", envs)

    t:= time.Now()
    zone_name, offset:= t.Zone() 
    fmt.Printf("        The zone name is: %s\n", zone_name) 
    fmt.Printf("        The zone offset returned is: %d\n", offset/3600)
    if offset/3600 != 8 {
        color.Red("        The time zone is not +0800!!  ")
    }else{
        color.Green("        The time zone is +0800  PASS")
    }

    var rLimit syscall.Rlimit
    err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)

    if err != nil {
        color.Red("Error Getting Rlimit ", err)
    }
    rlimit, err :=  strconv.ParseInt(strings.TrimSpace(string(rLimit.Cur)), 10, 32)
    if rlimit < 65535 {
        color.Red("        file no limit: %v\n",rLimit.Cur)
    }else{
        color.Green("        file no limit: %v\n",rLimit.Cur)

    }



    hostname, found := syscall.Getenv("HOSTNAME")
    if found {
        fmt.Printf("        HostName: %v\n", hostname)
    }else{
        fmt.Printf("        ERROR: hostname not found!")
    }

    fh, err := os.Open("/etc/pam.d/system-auth")
    if err != nil {
        panic(err)
    }

    r := bufio.NewReader(fh)
    for {
        line, err := r.ReadString('\n')
        line = strings.TrimSpace(line)
        if err != nil && err != io.EOF {
            panic(err)
        }
        if err == io.EOF {
            break
        }
        if strings.Contains(line,"password")  &&  strings.Contains(line,"requisite"){
            fmt.Println(line)
        }
    }

//      max user processes  
    cmd := exec.Command("bash","-c","ulimit -u")
    out, err := cmd.CombinedOutput()
    if err != nil {
	color.Red("cmd.Run() failed with %s\n", err)
    }
    mup, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 32)
    if err != nil {
        color.Red("        Max user processes convert int fail!! with error: %v ", err)
    }
    if  mup < 32760 {
        color.Red("        Max user processes is too small! ")
    }else{
        color.Green("        Max user processes is OK! ")
    } 
    fmt.Printf("        max user processes: %v\n", string(out))
  
//      openssl version 
    cmd = exec.Command("openssl","version")
    out, err = cmd.CombinedOutput()
    if err != nil {
        color.Red("openssl exec error with %s\n", err)
    }else{
        color.Green("        openssl version grater than 1.0.1e\n")
    }
    ov := string(out)
    if ov > "OpenSSL 1.0.1e" {
        color.Green("        openssl version: %v\n", string(out))
    }else{
        color.Red("        openssl version: %v\n", string(out))
    }


//   ssh connection time out
    cmd = exec.Command("grep","TMOUT","/etc/profile")
    out, err = cmd.CombinedOutput()
    if err != nil {
        color.Red("        SSH Connection timeout not set! \n")
    }
    re := regexp.MustCompile("TMOUT=([0-9]+)")
    tmout := re.FindStringSubmatch(string(out)) 
    color.Yellow("tmout: %v",tmout[1] )
    if (tmout != nil && string(tmout[1]) >= "300") {
        color.Green("        SSH Connection timeout: %v\n", string(tmout[1]))

    }else{
        color.Red("        SSH Connection timeout less than 300! \n")
    }
    
    fmt.Printf("        SSH Connection timeout: %v\n", string(out))

    // procce  weblogic
    color.Yellow("Checking weblogic=====")

    processes, _ := process.Processes()
    weblogic_found := false
    for _, proc := range processes {
        if err != nil {
            color.Red("        get Process info error with %v", err)
        }
        proccmd, _ := proc.Cmdline()
        exestr, _ := proc.Exe()
        procexe  := strings.Split(exestr,"/")
        if procexe[len(procexe) - 1]=="java" && strings.Contains(strings.ToLower(proccmd),"weblogic") { 
            color.Yellow("        %v \n",proccmd)
            check_weblogic(proc)
            weblogic_found = true
        }
    }
    if weblogic_found {
        color.Green("        weblogic check finished!\n")
    }else{
        color.Red("        weblogic not found!\n")
    }
    wl_home := os.Getenv("WL_HOME")
    fmt.Printf("%v \n",wl_home)
    fmt.Printf("%v \n",get_ip())


//  check redis

    redis_found := false
    for _, proc := range processes {
        if err != nil {
            color.Red("        get Process info error with %v", err)
        }
        proccmd, _ := proc.Cmdline()
        if len(proccmd) == 0 {  continue   }
        exestr := strings.Split(proccmd," ")[0]
        exestr1 := strings.Split(string(exestr),"/")
        procexe  := string(exestr1[len(exestr1)-1])
        if  "redis-server" == procexe {
            check_redis(proc)
            redis_found = true
        }
    }
    if redis_found {
    
    }
}
 
func get_ip()(ip string) {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        color.Red("get host ip error with: %v ",err)
        return
    }
    for _, address := range addrs {
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                ip := ipnet.IP.String()
                return ip
            }
        }
    }
    return ""
}

func check_weblogic ( proc *process.Process){

    cmdline, _ := proc.Cmdline()
    fmt.Println("%v", cmdline)
    // xms
    //xmsre := regexp.MustCompile("xms[0-9]+[m,g])")
    //xms:= xmsre.FindStringSubmatch(strings.ToLower(cmdline))[1]


    // xmx
    xmxre := regexp.MustCompile("xmx([0-9]+[m,g])")
    xmx := xmxre.FindStringSubmatch(strings.ToLower(cmdline))

    // permsize
    //permre := regexp.MustCompile(":permsize=[0-9]+[m,g])")
    //permsize:= permre.FindStringSubmatch(strings.ToLower(cmdline))[1]

    // maxpermsize
    maxpermre := regexp.MustCompile(":maxpermsize=([0-9]+[m,g])")
    maxpermsize := maxpermre.FindStringSubmatch(strings.ToLower(cmdline))

    if len(xmx) == 0 {
        color.Red("        weblogic Xmx not set !")
    }else{
        xmxstr := string(xmx[len(xmx)-1])
        switch xmxstr[len(xmxstr)-1:] { 
            case "m":
                if  size, _ :=strconv.ParseInt(xmxstr[0:len(xmxstr)-1], 10, 32); size < 6144 {
                    color.Red("        weblogic Xmx only %v , please set it up to 6144M.\n",xmxstr)
                }else{
                    color.Green("        weblogic Xmx is %v , big enough.\n",xmxstr)
                }

            case "g":
                if  size, _ := strconv.ParseInt(xmxstr[0:len(xmxstr)-1], 10, 32); size < 6 {
                    color.Red("        weblogic Xmx only %v , please set it up to 6G.\n",xmxstr)
                }else{
                    color.Green("        weblogic Xmx is %v , big enough.\n",xmxstr)
                }

            default:
                color.Red("        Parse Xmx parameter fail!")
        }
    }

    if len(maxpermsize) > 0 {
        maxpermsizestr := string(maxpermsize[len(maxpermsize)-1])
        switch  maxpermsizestr[len(maxpermsizestr)-1:] {
            case "m":
                if  size, _ := strconv.ParseInt(maxpermsizestr[0:len(maxpermsizestr)-1], 10, 32); size < 1024 {
                    color.Red("        weblogic MaxPermSize only %v , please set it up to 1024M.\n", maxpermsizestr)
                }else{
                    color.Green("        weblogic MaxPermSize s %v , big enough.\n", maxpermsizestr)
                }
    
            case "g":
                if  size, _ := strconv.ParseInt(maxpermsizestr[0:len(maxpermsizestr)-1], 10, 32 ); size < 1 {
                    color.Red("        weblogic MaxPermSize only %v , please set it up to 1G.\n", maxpermsizestr)
                }else{
                    color.Green("        weblogic MaxPermSize  is %v , big enough.\n", maxpermsizestr)
                }
    
            default:
                color.Red("        Parse MaxPermSize parameter fail!")
        }
    }else{
        color.Red("        weblogic MaxPermSize not set !")

    }
    minpoolsize_re := regexp.MustCompile("minpoolsize=([0-9]+)")
    minpoolsize := minpoolsize_re.FindStringSubmatch(strings.ToLower(cmdline))

    if len(minpoolsize) > 0 {
        minpoolsizeint, _ := strconv.ParseInt(minpoolsize[1], 10, 32)
        if minpoolsizeint < 500 {
            color.Red("        weblogic MinPoolSize less than 500 or not set, please set up to 500.\n")  
        }else{
            color.Green("        weblogic MinPoolSize is %v PASS.\n", minpoolsizeint )  
        }
    }else{
        color.Red("        weblogic MinPoolSize is not set !")
    }
    maxpoolsize_re := regexp.MustCompile("MaxPoolSize=([0-9]+)")
    maxpoolsize := maxpoolsize_re.FindStringSubmatch(strings.ToLower(cmdline))

    if len(maxpoolsize) > 0 {
        maxpoolsizeint, _ := strconv.ParseInt(maxpoolsize[1], 10, 32)
        if maxpoolsizeint < 1500 {
            color.Red("        weblogic MaxPoolSize less than 1500 or not set, please set up to 1500.\n")
        }else{
            color.Green("        weblogic MaxPoolSize is %v PASS.\n", maxpoolsizeint )

        }
    }else{
        color.Red("        weblogic MaxPoolSize is not set !")
    }

    if strings.Contains(strings.ToLower(cmdline),"djava.awt.headless=true")  {
        color.Green("        weblogic parameter java.awt.headless=true is configured PASS.\n" )
    }else{
        color.Red("        weblogic parameter java.awt.headless=true is not set.FAIL.\n" )

    }
}

func check_redis (proc *process.Process) (){
   connections, _ := proc.Connections()
   for _, connection := range connections {
       color.Magenta(" ip   %v", connection.Laddr.IP)
   }
   conn, err := redis.Dial("tcp", "127.0.0.1:6379")
   if err != nil {
      color.Red("redis connect error with: %n\r", err)
   }
   defer conn.Close()
}
