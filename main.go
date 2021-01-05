package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

const progName string = "suse-hacollect"
const ver string = "1.0.0"

func main() {

	// Handle command-line arguments
	c := Config{}
	c.Setup()
	flag.Parse()
	if c.version == true {
		fmt.Println(progName + " v" + ver + " (https://github.com/rfparedes/suse-hacollect)")
		return
	}
	d, err := time.Parse("2006-01-02", c.fromDate)
	t := time.Now()
	if err != nil || d.Format("2006-01-02") > t.Format("2006-01-02") {
		fmt.Println("From date entered is not valid")
		fmt.Println("example: suse-hacollect -f 2020-11-22")
		fmt.Println("")
		flag.PrintDefaults()
		return
	}

	// Create tmpdir first so we can start logging to it
	tmpDir := createTmpDir()

	// Log to file
	f, err := os.OpenFile(tmpDir+"/hacollect.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Print("Error opening logfile: ", err)
	}
	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)

	log.Print("Created tmpdir " + tmpDir)
	header()
	localHostname := getLocalHostname()
	log.Print("Local hostname is " + localHostname)
	installPkgs(false, localHostname)
	runSupportconfig(false, localHostname, tmpDir)
	runHbReport(localHostname, tmpDir, c.fromDate)
	remoteHostname, err := getRemoteHostname(tmpDir, localHostname)
	if err != nil {
		log.Print("Cannot get data from unknown remote host")
	} else {
		installPkgs(true, remoteHostname)
		log.Print("Remote Hostname is " + remoteHostname)
		runSupportconfig(true, remoteHostname, tmpDir)
	}
	f.Close()
	report(tmpDir, c.caseNum, c.upload)
	footer()
}

// Function to compress data
func compress(desc string, dst string, srcDir string, src string) (success int) {
	tar, err := exec.LookPath("tar")
	if err != nil {
		log.Print("Cannot find 'tar' executable.")
		log.Print("Cannot compress " + desc)
		return 1
	}
	cmd := exec.Command(tar, "Jcf", dst, "-C", srcDir, src)
	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Compressing " + desc + " "
	s.Color("fgGreen")
	s.Start()
	err = cmd.Run()
	if err != nil {
		log.Print(desc + " failed")
		s.Stop()
		return 1
	}
	s.Stop()
	log.Print("Compression of " + desc + " complete")
	return 0
}

// Function to create a temporary directory
func createTmpDir() string {
	const prefix = progName + "-"
	dir, err := ioutil.TempDir("/tmp", prefix)
	if err != nil {
		log.Fatal("Failed to create temp dir: ", err)
	}
	return dir
}

// Function to display footer
func footer() {
	log.Print("Report bugs/features at https://github.com/rfparedes/suse-hacollect/issues")
}

// Function to get local hostname
func getLocalHostname() (name string) {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown" // Return "unknown" as the hostname if error
	}
	return hostname
}

// Function to get remote hostname
func getRemoteHostname(tmpDir string, localHostname string) (name string, err error) {
	f, err := os.Open(tmpDir + "/members.txt")
	if err != nil {
		log.Print("Failed to open members.txt file: Cannot get remote hostname")
		return "", err
	}
	defer f.Close()
	line := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line = scanner.Text()
		break
	}
	s := strings.Split(line, " ")
	if s[0] == localHostname {
		return s[1], err
	}
	return s[0], err
}

// Function to print header
func header() {
	log.Print("Starting " + progName + " v" + ver)
}

// Function to install packages
func installPkgs(remote bool, hostname string) {
	var installCmd *exec.Cmd
	if remote == true {
		// Lookup ssh path
		ssh, err := exec.LookPath("ssh")
		if err != nil {
			log.Print("Cannot find 'ssh' executable")
			log.Print("Cannot install package on " + hostname)
			return
		}
		installCmd = exec.Command(ssh, "root@"+hostname, "zypper", "-q", "--non-interactive", "in", "supportutils-plugin-ha-sap")
	} else {
		// Lookup zypper path
		zypper, err := exec.LookPath("zypper")
		if err != nil {
			log.Print("Cannot find 'zypper' executable")
			log.Print("Cannot install package on " + hostname)
			return
		}
		installCmd = exec.Command(zypper, "-q", "--non-interactive", "in", "supportutils-plugin-ha-sap")
	}
	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Installing supportutils-plugin-ha-sap on " + hostname + " "
	s.Color("fgGreen")
	s.Start()
	err := installCmd.Run()
	if err != nil {
		s.Stop()
		log.Print("Install of supportutils-plugin-ha-sap on " + hostname + " FAILED")
		return
	}
	s.Stop()
	log.Print("Install of supportutils-plugin-ha-sap on " + hostname + " complete")
}

// Function to run hbreport
func runHbReport(hostname string, tmpDir string, startdate string) {
	// Lookup hb_report path and run it
	hbReport, err := exec.LookPath("hb_report")
	if err != nil {
		log.Print("Cannot find 'hb_report' executable")
		log.Print("Cannot run hb_report on " + hostname)
		return
	}
	cmd := exec.Command(hbReport, "-d", "-v", "-u", "root", "-f", startdate, tmpDir+"/hb_report")
	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Running hb_report from " + hostname + " "
	s.Color("fgGreen")
	s.Start()
	err = cmd.Run()
	if err != nil {
		log.Print("hb_report failed")
		s.Stop()
		return
	}
	s.Stop()
	log.Print("hbreport complete")

	// Copy memberFile to be used to identify other node
	srcFile, _ := os.Open(tmpDir + "/hb_report" + "/" + hostname + "/members.txt")
	defer srcFile.Close()
	dstFile, _ := os.OpenFile(tmpDir+"/members.txt", os.O_RDWR|os.O_CREATE, 0666)
	defer dstFile.Close()
	io.Copy(dstFile, srcFile)

	// Compress hb_report
	success := compress("hb_report", tmpDir+"/hb_report.txz", tmpDir, "hb_report")
	if success == 0 {
		os.RemoveAll(tmpDir + "/hb_report")
	}
}

// Function to finally compress and report
func report(tmpDir string, caseNum string, upload bool) {
	var filename string
	dst := "/var/log/"
	currentDate := time.Now()
	dateExt := currentDate.Format("01022006_150405")
	if len(caseNum) > 0 {
		filename = progName + "-" + "case" + caseNum + "-" + dateExt + ".txz"
	} else {
		filename = progName + "-" + dateExt + ".txz"
	}
	absFilename := dst + filename
	success := compress("all data", absFilename, filepath.Dir(tmpDir), filepath.Base(tmpDir))
	if success == 0 {
		os.RemoveAll(tmpDir)
	}
	log.Print("All Finished. File location: " + absFilename)
	if upload == true {
		uploadSuse(filename, absFilename)
	}
}

// Function to run supportconfig
func runSupportconfig(remote bool, hostname string, tmpDir string) {
	supportconfig, err := exec.LookPath("supportconfig")
	if err != nil {
		log.Print("Cannot find 'supportconfig' executable.")
		log.Print("Cannot run supportconfig on " + hostname)
		return
	}
	cmd := &exec.Cmd{}
	if remote == true {
		ssh, err := exec.LookPath("ssh")
		if err != nil {
			log.Print("Cannot find 'ssh' executable.")
			log.Print("Cannot run supportconfig on " + hostname)
			return
		}
		cmd = exec.Command(ssh, "root@"+hostname, supportconfig, "-Q", "-w", "-l", "-B", hostname+"-supportconfig", "-t", tmpDir)
	} else {
		cmd = exec.Command(supportconfig, "-Q", "-w", "-l", "-B", hostname+"-supportconfig", "-t", tmpDir)
	}
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	done := make(chan struct{})
	scanner := bufio.NewScanner(stdout)

	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Running supportconfig on " + hostname + " "
	s.Color("fgGreen")
	s.Start()

	// goroutine to inspect the supportconfig wait trace logging for any hangs
	go func() {
		timer := time.NewTimer(time.Second * 120)
		pgrepCmd := "pgrep supportconfig | xargs pgrep -n -P"
		defer timer.Stop()
		// Read line by line and process it
		for scanner.Scan() {
			// Reset timer after each new supportconfig function starting
			timer.Reset(time.Second * 120)
			go func() {
				<-timer.C
				var cpid []byte
				killCmd := &exec.Cmd{}
				hangMsg := "Cannot kill hanging supportconfig function. Manually terminate on " + hostname
				pkill, err := exec.LookPath("pkill")
				if err != nil {
					log.Print("Cannot find 'pkill' executable.")
					log.Print(hangMsg)
					return
				}
				if remote == true {
					ssh, err := exec.LookPath("ssh")
					if err != nil {
						log.Print("Cannot find 'ssh' executable.")
						log.Print(hangMsg)
						return
					}
					cpid, _ = exec.Command(ssh, "root@"+hostname, "bash", "-c", pgrepCmd).Output()
				} else {
					cpid, _ = exec.Command("bash", "-c", pgrepCmd).Output()
				}
				cpidToStr := strings.Replace(string(cpid), "\n", "", -1)
				if remote == true {
					ssh, err := exec.LookPath("ssh")
					if err != nil {
						log.Print("Cannot find 'ssh' executable.")
						log.Print(hangMsg)
						return
					}
					killCmd = exec.Command(ssh, "root@"+hostname, pkill, "-9", "-P", cpidToStr)
					err = killCmd.Run()
					if err != nil {
						log.Print(hangMsg)
						return
					}
				} else {
					killCmd = exec.Command(pkill, "-9", "-P", cpidToStr)
					err = killCmd.Run()
					if err != nil {
						log.Print(hangMsg)
						return
					}
				}
			}()
			scanner.Text()
		}
		// We're all done, unblock the channel
		done <- struct{}{}
	}()
	err = cmd.Run()
	if err != nil {
		log.Print("Supportconfig failed on " + hostname)
		s.Stop()
		return
	}
	<-done
	s.Stop()
	log.Print("Supportconfig on " + hostname + " complete")

	if remote == true {
		scp, err := exec.LookPath("scp")
		if err != nil {
			log.Print("Cannot find 'scp' executable.")
			log.Print("Cannot copy supportconfig from " + tmpDir + " on " + hostname)
			return
		}
		scpCmd := exec.Command(scp, "-q", "-r", "root@"+hostname+":"+tmpDir+"/*", tmpDir+"/")
		err = scpCmd.Run()
		if err != nil {
			log.Print("Cannot scp supportconfig from " + tmpDir + " on " + hostname)
			return
		}
		ssh, err := exec.LookPath("ssh")
		if err != nil {
			log.Print("Cannot find 'ssh' executable.")
			log.Print("Cannot remove " + tmpDir + " on " + hostname)
			return
		}
		sshCmd := exec.Command(ssh, "root@"+hostname, "rm", "-Rf", tmpDir)
		err = sshCmd.Run()
		if err != nil {
			log.Print("Cannot remove " + tmpDir + " on " + hostname)
			return
		}
	}

	// Compress supportconfig
	scDir := "scc_" + hostname + "-supportconfig"
	success := compress("supportconfig", tmpDir+"/"+hostname+"-supportconfig.txz", tmpDir, scDir)
	if success == 0 {
		os.RemoveAll(tmpDir + "/" + scDir)
	}
}

// Function to upload data to SUSE
func uploadSuse(filename string, absFilename string) {
	curl, err := exec.LookPath("curl")
	if err != nil {
		log.Print("Cannot find 'curl' executable.")
		log.Print("Cannot upload file. Please manually provide.")
		return
	}
	cmd := exec.Command(curl, "--connect-timeout", "30", "-L", "-A", "SupportConfig", "-T", absFilename, "https://support-ftp.us.suse.com/incoming/upload.php?file="+filename)
	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = "Uploading " + absFilename + " to SUSE over SSL"
	s.Color("fgGreen")
	s.Start()
	err = cmd.Run()
	if err != nil {
		log.Print("Upload to SUSE FAILED")
		s.Stop()
		return
	}
	s.Stop()
	log.Print("Upload to SUSE complete")
}
