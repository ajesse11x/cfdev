package acceptance

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os/exec"

	"fmt"

	"path"

	"time"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("hyperkit starts and telemetry", func() {
	var (
		cacheDir string
		session  *gexec.Session
		err      error
	)

	BeforeEach(func() {
		cacheDir = filepath.Join(cfdevHome, "cache")

		if os.Getenv("CFDEV_PLUGIN_PATH") == "" {
			SetupDependencies(cacheDir)
			os.Setenv("CFDEV_SKIP_ASSET_CHECK", "true")
		}

		os.RemoveAll(filepath.Join(cfdevHome, "analytics"))
	})

	AfterEach(func() {
		if IsWindows() {
			exec.Command("powershell.exe", "-Command", "Stop-Process -Name cfdev,cf -Force -EA 0").Run()
		}

		session.Kill()
	})

	It("optout", func() {
		cmd := exec.Command(GetCfPluginPath(), "dev", "start")
		inWriter, _ := cmd.StdinPipe()
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session, 10, 1).Should(gbytes.Say("Are you ok with CF Dev periodically capturing anonymized telemetry"))

		fmt.Fprintln(inWriter, "no")

		Eventually(func() ([]byte, error) {
			return ioutil.ReadFile(filepath.Join(cfdevHome, "analytics", "analytics.txt"))
		}, 10, 1).Should(MatchJSON(`{"enabled":false, "props":{"type":"cf"}}`))

	})

	It("optin", func() {
		cmd := exec.Command(GetCfPluginPath(), "dev", "start")
		inWriter, _ := cmd.StdinPipe()
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session).Should(gbytes.Say("Are you ok with CF Dev periodically capturing anonymized telemetry"))

		fmt.Fprintln(inWriter, "yes")

		Eventually(func() ([]byte, error) {
			return ioutil.ReadFile(filepath.Join(cfdevHome, "analytics", "analytics.txt"))
		}).Should(MatchJSON(`{"enabled":true, "props":{"type":"cf"}}`))
	})

	It("is already opted in", func() {
		err := os.MkdirAll(path.Join(cfdevHome, "analytics"), 0755)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(path.Join(cfdevHome, "analytics", "analytics.txt"), []byte("optin"), 0755)
		Expect(err).ToNot(HaveOccurred())

		cmd := exec.Command(GetCfPluginPath(), "dev", "start")
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Consistently(session, time.Second).ShouldNot(gbytes.Say("Are you ok with CF Dev periodically capturing anonymized telemetry"))

		Expect(ioutil.ReadFile(filepath.Join(cfdevHome, "analytics", "analytics.txt"))).Should(MatchJSON([]byte(`{"enabled":true, "props":{"type":"cf"}}`)))
	})

	It("allows noninteractive telemetry --off command", func() {
		cmd := exec.Command(GetCfPluginPath(), "dev", "telemetry", "--off")
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Consistently(session, time.Second).ShouldNot(gbytes.Say("Are you ok with CF Dev periodically capturing anonymized telemetry"))

		Expect(ioutil.ReadFile(filepath.Join(cfdevHome, "analytics", "analytics.txt"))).Should(MatchJSON([]byte(`{"enabled":false, "props":{}}`)))
	})

	It("allows noninteractive telemetry --on command", func() {
		cmd := exec.Command(GetCfPluginPath(), "dev", "telemetry", "--on")
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Consistently(session, time.Second).ShouldNot(gbytes.Say("Are you ok with CF Dev periodically capturing anonymized telemetry"))

		Expect(ioutil.ReadFile(filepath.Join(cfdevHome, "analytics", "analytics.txt"))).Should(MatchJSON([]byte(`{"enabled":true, "props":{}}`)))
	})
})
