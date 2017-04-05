package integration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"code.cloudfoundry.org/silk/controller/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Silk Controller", func() {

	var (
		session        *gexec.Session
		conf           config.Config
		configFilePath string
		baseURL        string
	)

	BeforeEach(func() {
		conf = config.Config{
			ListenHost:      "127.0.0.1",
			ListenPort:      50000 + GinkgoParallelNode(),
			DebugServerPort: 60000 + GinkgoParallelNode(),
		}
		baseURL = fmt.Sprintf("http://%s:%d", conf.ListenHost, conf.ListenPort)

		configFile, err := ioutil.TempFile("", "config-file-")
		Expect(err).NotTo(HaveOccurred())
		configFilePath = configFile.Name()
		Expect(configFile.Close()).To(Succeed())
		Expect(conf.WriteToFile(configFilePath)).To(Succeed())

		cmd := exec.Command(controllerBinaryPath, "-config-file", configFilePath)
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the http server to boot")
		serverIsUp := func() error { return VerifyHTTPConnection(baseURL) }
		Eventually(serverIsUp, DEFAULT_TIMEOUT).Should(Succeed())
	})

	AfterEach(func() {
		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		Expect(os.Remove(configFilePath)).To(Succeed())
	})

	It("gracefully terminates when sent an interrupt signal", func() {
		Consistently(session).ShouldNot(gexec.Exit())

		session.Interrupt()
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	})

	It("runs the cf debug server on the configured port", func() {
		resp, err := http.Get(
			fmt.Sprintf("http://127.0.0.1:%d/debug/pprof", conf.DebugServerPort),
		)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})