package main

import (
	"os"
	"fmt"
	"time"
	"regexp"
	"errors"
	"strings"
	"syscall"

	"os/exec"
	"io/ioutil"
)

var filepath string
var chanl chan bool
var echanl chan error

func Log(service string, loglevel string, message string) error {
        file, err := os.OpenFile("/var/log/nfschk.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
                return errors.New("Failed to open log file for writing: " + err.Error())
        }
        defer file.Close()

        current_time := time.Now().Local()
        t := current_time.Format("Jan 02 2006 03:04:05")
        _, err = file.WriteString(loglevel + " | " + t + " | " + service + " | " + message + "\n")

        if err != nil {
                return errors.New("Failed to write to log file: " + err.Error())
        }

        return nil
}

func CheckNFS(chanl chan bool, echanl chan error) {
	if CheckISMountedNFS() {
		err := ioutil.WriteFile(filepath + "/testfile.dat", []byte("test from nfschk\n"), 0777)
		if err != nil {
			chanl <- false
		} else {
			os.Remove(filepath + "/testfile.dat")
			chanl <- true
		}
	} else {
		Log("NFSCheck", "INFO", "NFS not mounted")
		echanl <- errors.New("NFS not mounted")
	}

	return 
}

func CheckISMountedNFS() bool {
	var foundflag bool
	var ffoundflag bool

	foundflag = false
	ffoundflag = false

	dat, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		Log("NFSCheck", "INFO", "Error Reading /proc/mounts: " + err.Error())
		fmt.Println("Error: " + err.Error())
		return false
	}

	entire := string(dat)
	lines := strings.Split(entire, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		path := parts[1]
		typ := parts[2]

		if path == filepath || path + "/" == filepath && typ == "nfs" {
			foundflag = true
		}
	}

	fdat, err := ioutil.ReadFile("/etc/fstab")
	if err != nil {
		Log("NFSCheck", "INFO", "Error Reading /etc/fstab: " + err.Error())
		fmt.Println("Error: " + err.Error())
		return false
	}

	fentire := string(fdat)
	flines := strings.Split(fentire, "\n")
	for _, fline := range flines {
		if strings.HasPrefix(fline, "#") {
			continue
		}

		fline = strings.Replace(fline, "\t", " ", -1)
		re_leadclose_whtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
		re_inside_whtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
		final := re_leadclose_whtsp.ReplaceAllString(fline, "")
		final = re_inside_whtsp.ReplaceAllString(final, " ")

		if fline == "" || final == "" {
			continue
		}

		parts := strings.Split(final, " ")
		path := parts[1]
		typ := parts[2]

		if path == filepath || path + "/" == filepath && typ == "nfs" {
			ffoundflag = true
		}
	}

	if ffoundflag && ! foundflag {
		return false
	}

	if ffoundflag && foundflag {
		return true
	}

	return false
}

func CheckIfNFS() bool {
	var foundflag bool
	var ffoundflag bool
	var noautoflag bool

	foundflag = false
	ffoundflag = false
	noautoflag = false

	dat, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		Log("NFSCheck", "INFO", "Error Reading /proc/mounts: " + err.Error())
		fmt.Println("Error: " + err.Error())
		return false
	}

	entire := string(dat)
	lines := strings.Split(entire, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		path := parts[1]
		typ := parts[2]

		if path == filepath || path + "/" == filepath && typ == "nfs" {
			foundflag = true
		}
	}

	fdat, err := ioutil.ReadFile("/etc/fstab")
	if err != nil {
		Log("NFSCheck", "INFO", "Error Reading /etc/fstab: " + err.Error())
		fmt.Println("Error: " + err.Error())
		return false
	}

	fentire := string(fdat)
	flines := strings.Split(fentire, "\n")
	for _, fline := range flines {
		if strings.HasPrefix(fline, "#") {
			continue
		}

		fline = strings.Replace(fline, "\t", " ", -1)
		re_leadclose_whtsp := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
		re_inside_whtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
		final := re_leadclose_whtsp.ReplaceAllString(fline, "")
		final = re_inside_whtsp.ReplaceAllString(final, " ")

		if fline == "" || final == "" {
			continue
		}

		parts := strings.Split(final, " ")
		path := parts[1]
		typ := parts[2]
		opt := parts[3]

		if strings.Contains(opt, "noauto") {
			noautoflag = true
		}

		if path == filepath || path + "/" == filepath && typ == "nfs" {
			ffoundflag = true
		}
	}

	if noautoflag {
		Log("NFSCheck", "INFO", "Found noauto In NFS Options.  Refusing To Remount")
		fmt.Println("found noauto")
		return false
	}

	if foundflag {
		return true
	}

	if ffoundflag {
		return true
	}

	return false 

}

func UmountNFS(chanl chan bool) {
	var err error
	var waitStatus syscall.WaitStatus

	cmd := exec.Command("/usr/bin/umount", "-l", filepath)
	if err = cmd.Run(); err != nil {
		Log("NFSCheck", "INFO", "Error Running Umount: " + err.Error())
		chanl <- false
		return
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		waitStatus = exitError.Sys().(syscall.WaitStatus)
	}

	if waitStatus != 0 {
		chanl <- false
		return
	}

	chanl <- true
	return
}

func MountNFS(chanl chan bool) {
	var err error
	var waitStatus syscall.WaitStatus

	cmd := exec.Command("/usr/bin/mount", "-av")
	if err = cmd.Run(); err != nil {
		Log("NFSCheck", "INFO", "Error Running Mount: " + err.Error())
		chanl <- false
		return
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		waitStatus = exitError.Sys().(syscall.WaitStatus)
	}

	if waitStatus != 0 {
		chanl <- false
		return
	}

	chanl <- true
	return
}

func main() {
	var status bool
	var estatus bool
	var umstatus bool
	var mstatus bool
	chanl = make(chan bool)
	echanl = make(chan error)

	status = false
	estatus = false
	mstatus = false
	umstatus = false

	if len(os.Args) < 2 {
		fmt.Println("Usage: " + os.Args[0] + " <path to nfs directory>")
		return
	}

	filepath = os.Args[1]

	isNFS := CheckIfNFS()
	if ! isNFS {
		Log("NFSCheck", "INFO", "This Path Doesn't Appear To Be An NFS Mount.  Please Check /etc/fstab and Ensure This Path Is there, And That noauto is NOT Set")
		fmt.Println("This Path Doesn't Appear To Be An NFS Mount.")
		fmt.Println("Please Check /etc/fstab And Ensure This Path Is There, And That noauto is NOT Set")
		os.Exit(1)
	}

	go CheckNFS(chanl, echanl)
	select {
		case flagval := <-chanl:
			if flagval {
				status = true
			}
		case <-echanl:
			estatus = true
		case <-time.After(3 * time.Second):
	}

	if estatus {
		retrystatus := false
		Log("NFSCheck", "WARN", "NFS Not Mounted, But Should Be... Trying To Mount")
		go MountNFS(chanl)
		select {
			case flagval := <-chanl:
				if flagval {
					retrystatus = true
				}
			case <-time.After(3 * time.Second):
		}

		if ! retrystatus {
			Log("NFSCheck", "ERROR", "Failed To Mount " + filepath)
			fmt.Println("Failed To Mount: " + filepath)
			os.Exit(2)
		}

		Log("NFSCheck", "WARN", "NFS Was Not Mounted, But It Is Now For: " + filepath)
		fmt.Println("NFS Had To Be Remounted: " + filepath)
		os.Exit(1)
	}

	if ! status {
		go UmountNFS(chanl)
		select {
			case flagval := <-chanl:
				if flagval {
					umstatus = true
				}
			case <-time.After(3 * time.Second):
		}

		if ! umstatus {
			Log("NFSCheck", "ERROR", "Failed To Unmount " + filepath)
			fmt.Println("Failed To Umount: " + filepath)
			os.Exit(2)
		}

		go MountNFS(chanl)
		select {
			case flagval := <-chanl:
				if flagval {
					mstatus = true
				}
			case <-time.After(3 * time.Second):
		}

		if ! mstatus {
			Log("NFSCheck", "ERROR", "Failed To Mount " + filepath)
			fmt.Println("Failed To Mount: " + filepath)
			os.Exit(2)
		}

		Log("NFSCheck", "WARN", "Had To Remount NFS System " + filepath)
		fmt.Println("Remounted NFS System: " + filepath)
		os.Exit(1)
	} else {
		fmt.Println("All Good")
		os.Exit(0)
	}
}
