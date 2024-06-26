package main

import (
	"bytes"
	"log"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"regexp"
	"io"
	"time"
	"context"
	"flag"

)

const (
	localAddr = "/run/scgi_proxy.sock"
	remoteAddr = "/run/synoscgi.sock"
	configPath = "/etc/syno_cpuinfo/config.conf"
	sensorsConfigPath = "/etc/sensors3.conf"
)

type CpuInfo struct {
	Vendor   string
	Family   string
	Series   string
	Cores    string
	ClockSpeed    int
}

var cpuInfo CpuInfo


func replaceCPUInfo(data []byte) []byte {
    temp := getTempFromSensors()
    replacements := map[string]string{
        `"cpu_family":"[^"]+"`:  fmt.Sprintf(`"cpu_family":"%s"`, cpuInfo.Family),
        `"cpu_series":"[^"]+"`:  fmt.Sprintf(`"cpu_series":"%s"`, cpuInfo.Series),
        `"cpu_vendor":"[^"]+"`:  fmt.Sprintf(`"cpu_vendor":"%s"`, cpuInfo.Vendor),
        `"sys_temp":\d+`:        fmt.Sprintf(`"sys_temp":%d`, temp),
    }

	if cpuInfo.ClockSpeed != 0 {
		replacements[`"cpu_clock_speed":\d+`] = fmt.Sprintf(`"cpu_clock_speed":%d`, cpuInfo.ClockSpeed)
	}

	if cpuInfo.Cores != "" {
		replacements[`"cpu_cores":"\d+"`] = fmt.Sprintf(`"cpu_cores":"%s"`, cpuInfo.Cores)
	}

    for pattern, replacement := range replacements {
        re := regexp.MustCompile(pattern)
        data = re.ReplaceAll(data, []byte(replacement))
    }

    return data
}

func isIgnorableError(err error) bool {
	if netErr, ok := err.(net.Error); ok && !netErr.Temporary() {
		return true
	}
	if err == io.EOF {
		return true
	}
	return false
}

func handleConnection(localConn net.Conn, remoteAddr string) {
	defer localConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	remoteConn, err := net.DialTimeout("unix", remoteAddr, 5*time.Second)
	if err != nil {
		log.Printf("Failed to connect to remote socket: %v", err)
		return
	}
	defer remoteConn.Close()

	go transferData(ctx, localConn, remoteConn, cancel)
	go transferDataWithModification(ctx, remoteConn, localConn, cancel)

	<-ctx.Done()
}


func transferData(ctx context.Context, src, dst net.Conn, cancel context.CancelFunc) {
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := src.Read(buf)
			if n > 0 {
				if _, err := dst.Write(buf[:n]); err != nil {
					log.Printf("Error writing to destination: %v", err)
					cancel()
					return
				}
			}
			if err != nil {
				if !(isIgnorableError(err)) {
					log.Printf("Error reading from client source: %v", err)
				}
				cancel()
				return
			}
		}
	}
}


func transferDataWithModification(ctx context.Context, src, dst net.Conn, cancel context.CancelFunc) {
	buf := make([]byte, 32*1024)
	var dataBuffer bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := src.Read(buf)
			if n > 0 {
				data := buf[:n]
				if bytes.Contains(data, []byte(`"cpu_clock_speed"`)) {
					dataBuffer.Write(data)
					modifiedData := replaceCPUInfo(dataBuffer.Bytes())
					dataBuffer.Reset()
					if _, err := dst.Write(modifiedData); err != nil {
						log.Printf("Error writing modified data to destination: %v", err)
						cancel()
						return
					}
				} else {
					if _, err := dst.Write(data); err != nil {
						log.Printf("Error writing to destination: %v", err)
						cancel()
						return
					}
				}
			}
			if err != nil {
				if !(isIgnorableError(err)) {
					log.Printf("Error reading from remote source: %v", err)
				}
				cancel()
				return
			}
		}
	}
}

func listenAndProxy(localAddr, remoteAddr string) {
	if err := os.RemoveAll(localAddr); err != nil {
        log.Fatalf("Failed to remove existing socket file: %v", err)
    }

    localListener, err := net.Listen("unix", localAddr)
    if err != nil {
        log.Fatalf("Failed to listen on local socket: %v", err)
    }

    if err := os.Chmod(localAddr, 0777); err != nil {
        log.Fatalf("Failed to set permissions on socket file: %v", err)
    }
	log.Printf("Listening on %s, proxying to %s\n", localAddr, remoteAddr)
    defer os.Remove(localAddr)
    defer localListener.Close()

    for {
        localConn, err := localListener.Accept()
        if err != nil {
            log.Printf("Failed to accept local connection: %v", err)
            continue
        }
        go handleConnection(localConn, remoteAddr)
    }
}

func readAndReload(){
	readConfig(configPath)
	sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGHUP)

    go func() {
        for {
			sig := <-sigs
			if sig == syscall.SIGHUP {
				log.Println("Performing reload...")
				readConfig(configPath)
			}
        }
    }()
}


func main() {

	infoFlag := flag.Bool("i", false, "Read and print CPU info")
	tempFlag := flag.Bool("t", false, "Read and print CPU Temperature")
	flag.Parse()

	if *infoFlag {
		readConfig(configPath)
		os.Exit(0)
	} else if *tempFlag {
		if temp := getTempFromSensors(); temp !=0 {
			log.Printf("CPU Temperature: \033[32m%d\033[0m\n", temp)
			os.Exit(0)
		}else {
			log.Println("Error reading CPU Temperature")
			os.Exit(1)
		}
	}

	readAndReload()

	listenAndProxy(localAddr, remoteAddr)

}
