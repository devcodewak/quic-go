package integrationtests

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"strconv"

	"github.com/lucas-clemente/quic-go/h2quic"
	"github.com/lucas-clemente/quic-go/internal/utils"
	"github.com/lucas-clemente/quic-go/testdata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	dataLen      = 500 * 1024       // 500 KB
	dataLongLen  = 50 * 1024 * 1024 // 50 MB
	dlDataPrefix = "quic-go_dl_test_"
)

var (
	server         *h2quic.Server
	dataMan        dataManager
	port           string
	downloadDir    string
	clientPath     string
	serverPath     string
	nFilesUploaded int32

	logFileName string // the log file set in the ginkgo flags
	logFile     *os.File
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Tests Suite")
}

var _ = BeforeSuite(func() {
	setupHTTPHandlers()
	setupQuicServer()

	downloadDir = os.Getenv("HOME") + "/Downloads/"
})

// read the logfile command line flag
// to set call ginkgo -- -logfile=log.txt
func init() {
	flag.StringVar(&logFileName, "logfile", "", "log file")
}

var _ = BeforeEach(func() {
	// set custom time format for logs
	utils.SetLogTimeFormat("15:04:05.000")
	_, thisfile, _, ok := runtime.Caller(0)
	if !ok {
		Fail("Failed to get current path")
	}
	clientPath = filepath.Join(thisfile, fmt.Sprintf("../../../quic-clients/client-%s-debug", runtime.GOOS))
	serverPath = filepath.Join(thisfile, fmt.Sprintf("../../../quic-clients/server-%s-debug", runtime.GOOS))

	if len(logFileName) > 0 {
		var err error
		logFile, err = os.Create("./log.txt")
		Expect(err).ToNot(HaveOccurred())
		log.SetOutput(logFile)
		utils.SetLogLevel(utils.LogLevelDebug)
	}
})

var _ = JustBeforeEach(startQuicServer)

var _ = AfterEach(func() {
	stopQuicServer()

	if len(logFileName) > 0 {
		_ = logFile.Close()
	}

	nFilesUploaded = 0
})

var _ = AfterEach(func() {
	removeDownloadData()
})

func setupHTTPHandlers() {
	defer GinkgoRecover()

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		_, err := io.WriteString(w, "Hello, World!\n")
		Expect(err).NotTo(HaveOccurred())
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		data := dataMan.GetData()
		Expect(data).ToNot(HaveLen(0))
		_, err := w.Write(data)
		Expect(err).NotTo(HaveOccurred())
	})

	http.HandleFunc("/data/", func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		data := dataMan.GetData()
		Expect(data).ToNot(HaveLen(0))
		_, err := w.Write(data)
		Expect(err).NotTo(HaveOccurred())
	})

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		body, err := ioutil.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		_, err = w.Write(body)
		Expect(err).NotTo(HaveOccurred())
	})

	// Requires the len & num GET parameters, e.g. /uploadform?len=100&num=1
	http.HandleFunc("/uploadtest", func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		response := uploadHTML
		response = strings.Replace(response, "LENGTH", r.URL.Query().Get("len"), -1)
		response = strings.Replace(response, "NUM", r.URL.Query().Get("num"), -1)
		_, err := io.WriteString(w, response)
		Expect(err).NotTo(HaveOccurred())
	})

	http.HandleFunc("/uploadhandler", func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()

		l, err := strconv.Atoi(r.URL.Query().Get("len"))
		Expect(err).NotTo(HaveOccurred())

		defer r.Body.Close()
		actual, err := ioutil.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(bytes.Equal(actual, generatePRData(l))).To(BeTrue())

		atomic.AddInt32(&nFilesUploaded, 1)
	})
}

func startQuicServer() {
	server = &h2quic.Server{
		Server: &http.Server{
			TLSConfig: testdata.GetTLSConfig(),
		},
	}

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	Expect(err).NotTo(HaveOccurred())
	conn, err := net.ListenUDP("udp", addr)
	Expect(err).NotTo(HaveOccurred())
	port = strconv.Itoa(conn.LocalAddr().(*net.UDPAddr).Port)

	go func() {
		defer GinkgoRecover()
		server.Serve(conn)
	}()
}

<<<<<<< HEAD
func stopQuicServer() {
	Expect(server.Close()).NotTo(HaveOccurred())
}

func setupSelenium() {
	var err error
	pullCmd := exec.Command("docker", "pull", "lclemente/standalone-chrome:dev")
	pull, err := gexec.Start(pullCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	// Assuming a download at 10 Mbit/s
	Eventually(pull, 10*time.Minute).Should(gexec.Exit(0))

	dockerCmd := exec.Command(
		"docker",
		"run",
		"-i",
		"--rm",
		"-p=4444:4444",
		"--name", "quic-test-selenium",
		"lclemente/standalone-chrome:dev",
	)
	docker, err = gexec.Start(dockerCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(docker.Out, 10).Should(gbytes.Say("Selenium Server is up and running"))
}

func stopSelenium() {
	docker.Interrupt().Wait(10)
}

func getWebdriverForVersion(version protocol.VersionNumber) selenium.WebDriver {
	caps := selenium.Capabilities{
		"browserName": "chrome",
		"chromeOptions": map[string]interface{}{
			"args": []string{
				"--enable-quic",
				"--no-proxy-server",
				"--origin-to-force-quic-on=quic.clemente.io:443",
				fmt.Sprintf(`--host-resolver-rules=MAP quic.clemente.io:443 %s:%s`, GetLocalIP(), port),
				fmt.Sprintf(`--quic-version=QUIC_VERSION_%d`, version),
			},
		},
	}
	wd, err := selenium.NewRemote(caps, "http://localhost:4444/wd/hub")
	Expect(err).NotTo(HaveOccurred())
	return wd
}

func GetLocalIP() string {
	// First, try finding interface docker0
	i, err := net.InterfaceByName("docker0")
	if err == nil {
		var addrs []net.Addr
		addrs, err = i.Addrs()
		Expect(err).NotTo(HaveOccurred())
		return addrs[0].(*net.IPNet).IP.String()
	}

	addrs, err := net.InterfaceAddrs()
	Expect(err).NotTo(HaveOccurred())
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	panic("no addr")
}

func removeDownload(filename string) {
	cmd := exec.Command("docker", "exec", "-i", "quic-test-selenium", "rm", "-f", "/home/seluser/Downloads/"+filename)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 5).Should(gexec.Exit(0))
}

// getDownloadSize gets the file size of a file in the /home/seluser/Downloads folder in the docker container
=======
// getDownloadSize gets the file size of a file in the local download folder
>>>>>>> a9ecc2d... Replace docker with chromedp for integration tests
func getDownloadSize(filename string) int {
	stat, err := os.Stat(downloadDir + filename)
	if err != nil {
		return 0
	}
	return int(stat.Size())
}

// getDownloadMD5 gets the md5 sum file of a file in the local download folder
func getDownloadMD5(filename string) []byte {
	return getFileMD5(filepath.Join(downloadDir, filename))
}

func getFileMD5(filename string) []byte {
	var result []byte
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return nil
	}
	return hash.Sum(result)
}

func getRandomDlName() string {
	return dlDataPrefix + strconv.Itoa(time.Now().Nanosecond())
}

func removeDownloadData() {
	pattern := downloadDir + dlDataPrefix + "*"
	if len(pattern) < 10 || !strings.Contains(pattern, "quic-go") {
		panic("DL dir looks weird: " + pattern)
	}
	paths, err := filepath.Glob(pattern)
	Expect(err).NotTo(HaveOccurred())
	if len(paths) > 2 {
		panic("warning: would have deleted too many files, pattern " + pattern)
	}
	for _, path := range paths {
		err = os.Remove(path)
		Expect(err).NotTo(HaveOccurred())
	}
}

const uploadHTML = `
<html>
<body>
<script>
  var buf = new ArrayBuffer(LENGTH);
  var arr = new Uint8Array(buf);
  var seed = 1;
  for (var i = 0; i < LENGTH; i++) {
    // https://en.wikipedia.org/wiki/Lehmer_random_number_generator
    seed = seed * 48271 % 2147483647;
    arr[i] = seed;
  }
	for (var i = 0; i < NUM; i++) {
		var req = new XMLHttpRequest();
		req.open("POST", "/uploadhandler?len=" + LENGTH, true);
		req.send(buf);
	}
</script>
</body>
</html>
`

func generatePRData(l int) []byte {
	res := make([]byte, l)
	seed := uint64(1)
	for i := 0; i < l; i++ {
		seed = seed * 48271 % 2147483647
		res[i] = byte(seed)
	}
	return res
}
