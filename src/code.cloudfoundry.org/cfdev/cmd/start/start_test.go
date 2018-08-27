package start_test

import (
	"runtime"

	"code.cloudfoundry.org/cfdev/iso"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/cfdev/cfanalytics"
	"code.cloudfoundry.org/cfdev/cmd/start"
	"code.cloudfoundry.org/cfdev/cmd/start/mocks"
	"code.cloudfoundry.org/cfdev/config"
	"code.cloudfoundry.org/cfdev/provision"
	"code.cloudfoundry.org/cfdev/resource"
	"github.com/golang/mock/gomock"
	"code.cloudfoundry.org/cfdev/hypervisor"
)

var _ = Describe("Start", func() {

	var (
		mockController      *gomock.Controller
		mockUI              *mocks.MockUI
		mockAnalyticsClient *mocks.MockAnalyticsClient
		mockToggle          *mocks.MockToggle
		mockHostNet         *mocks.MockHostNet
		mockHost            *mocks.MockHost
		mockCache           *mocks.MockCache
		mockCFDevD          *mocks.MockCFDevD
		mockVpnKit          *mocks.MockVpnKit
		mockAnalyticsD      *mocks.MockAnalyticsD
		mockHypervisor      *mocks.MockHypervisor
		mockProvisioner     *mocks.MockProvisioner
		mockIsoReader       *mocks.MockIsoReader

		startCmd      start.Start
		exitChan      chan struct{}
		localExitChan chan string
		tmpDir        string
		cacheDir      string
		depsIsoPath   string
		metadata      iso.Metadata
	)

	BeforeEach(func() {
		var err error
		mockController = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI(mockController)
		mockAnalyticsClient = mocks.NewMockAnalyticsClient(mockController)
		mockToggle = mocks.NewMockToggle(mockController)
		mockHostNet = mocks.NewMockHostNet(mockController)
		mockHost = mocks.NewMockHost(mockController)
		mockCache = mocks.NewMockCache(mockController)
		mockCFDevD = mocks.NewMockCFDevD(mockController)
		mockVpnKit = mocks.NewMockVpnKit(mockController)
		mockAnalyticsD = mocks.NewMockAnalyticsD(mockController)
		mockHypervisor = mocks.NewMockHypervisor(mockController)
		mockProvisioner = mocks.NewMockProvisioner(mockController)
		mockIsoReader = mocks.NewMockIsoReader(mockController)

		localExitChan = make(chan string, 3)
		tmpDir, err = ioutil.TempDir("", "start-test-home")
		cacheDir = filepath.Join(tmpDir, "some-cache-dir")
		Expect(err).NotTo(HaveOccurred())

		startCmd = start.Start{
			Config: config.Config{
				CFDevHome:      tmpDir,
				StateDir:       filepath.Join(tmpDir, "some-state-dir"),
				VpnKitStateDir: filepath.Join(tmpDir, "some-vpnkit-state-dir"),
				CacheDir:       cacheDir,
				CFRouterIP:     "some-cf-router-ip",
				BoshDirectorIP: "some-bosh-director-ip",
				Dependencies: resource.Catalog{
					Items: []resource.Item{
						{Name: "some-item"},
						{Name: "cf-deps.iso"},
					},
				},
			},
			Exit:            exitChan,
			LocalExit:       localExitChan,
			UI:              mockUI,
			Analytics:       mockAnalyticsClient,
			AnalyticsToggle: mockToggle,
			HostNet:         mockHostNet,
			Host:            mockHost,
			Cache:           mockCache,
			CFDevD:          mockCFDevD,
			VpnKit:          mockVpnKit,
			AnalyticsD:      mockAnalyticsD,
			Hypervisor:      mockHypervisor,
			Provisioner:     mockProvisioner,
			IsoReader:       mockIsoReader,
		}

		depsIsoPath = filepath.Join(cacheDir, "cf-deps.iso")
		metadata = iso.Metadata{
			Version:       "v1",
			DefaultMemory: 8765,
			Services: []provision.Service{
				{
					Name:       "some-service",
					Handle:     "some-handle",
					Script:     "/path/to/some-script",
					Deployment: "some-deployment",
				},
				{
					Name:       "some-other-service",
					Handle:     "some-other-handle",
					Script:     "/path/to/some-other-script",
					Deployment: "some-other-deployment",
				},
			},
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		mockController.Finish()
	})

	Describe("Execute", func() {
		Context("when no args are provided", func() {
			// TODO test splashMessage
			It("starts the vm with default settings", func() {
				if runtime.GOOS == "darwin" {
					mockUI.EXPECT().Say("Installing cfdevd network helper...")
					mockCFDevD.EXPECT().Install()
				}

				gomock.InOrder(
					mockToggle.EXPECT().SetProp("type", "cf"),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
					mockHost.EXPECT().CheckRequirements(),
					mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),

					mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
					mockUI.EXPECT().Say("Downloading Resources..."),
					mockCache.EXPECT().Sync(resource.Catalog{
						Items: []resource.Item{
							{Name: "some-item"},
							{Name: "cf-deps.iso"},
						},
					}),
					mockIsoReader.EXPECT().Read(depsIsoPath).Return(metadata, nil),
					mockUI.EXPECT().Say("Creating the VM..."),
					mockHypervisor.EXPECT().CreateVM(hypervisor.VM{
						Name:     "cfdev",
						CPUs:     7,
						MemoryMB: 8765,
						DepsIso:  filepath.Join(cacheDir, "cf-deps.iso"),
					}),
					mockUI.EXPECT().Say("Starting VPNKit..."),
					mockVpnKit.EXPECT().Start(),
					mockVpnKit.EXPECT().Watch(localExitChan),
					mockUI.EXPECT().Say("Starting the VM..."),
					mockHypervisor.EXPECT().Start("cfdev"),
					mockUI.EXPECT().Say("Waiting for Garden..."),
					mockProvisioner.EXPECT().Ping(),
					mockUI.EXPECT().Say("Deploying the BOSH Director..."),
					mockProvisioner.EXPECT().DeployBosh(),
					mockUI.EXPECT().Say("Deploying CF..."),
					mockProvisioner.EXPECT().ReportProgress(mockUI, "cf"),
					mockProvisioner.EXPECT().DeployCloudFoundry(nil),
					mockProvisioner.EXPECT().DeployServices(mockUI, []provision.Service{
						{
							Name:       "some-service",
							Handle:     "some-handle",
							Script:     "/path/to/some-script",
							Deployment: "some-deployment",
						},
						{
							Name:       "some-other-service",
							Handle:     "some-other-handle",
							Script:     "/path/to/some-other-script",
							Deployment: "some-other-deployment",
						},
					}),

					mockToggle.EXPECT().Get().Return(true),
					mockAnalyticsD.EXPECT().Start(),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_END),
				)

				Expect(startCmd.Execute(start.Args{
					Cpus: 7,
					Mem:  0,
				})).To(Succeed())
			})

			It("starts the vm with analytics toggled off", func() {
				if runtime.GOOS == "darwin" {
					mockUI.EXPECT().Say("Installing cfdevd network helper...")
					mockCFDevD.EXPECT().Install()
				}

				gomock.InOrder(
					mockToggle.EXPECT().SetProp("type", "cf"),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
					mockHost.EXPECT().CheckRequirements(),
					mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),

					mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
					mockUI.EXPECT().Say("Downloading Resources..."),
					mockCache.EXPECT().Sync(resource.Catalog{
						Items: []resource.Item{
							{Name: "some-item"},
							{Name: "cf-deps.iso"},
						},
					}),
					mockIsoReader.EXPECT().Read(depsIsoPath).Return(metadata, nil),
					mockUI.EXPECT().Say("Creating the VM..."),
					mockHypervisor.EXPECT().CreateVM(hypervisor.VM{
						Name:     "cfdev",
						CPUs:     7,
						MemoryMB: 8765,
						DepsIso:  filepath.Join(cacheDir, "cf-deps.iso"),
					}),
					mockUI.EXPECT().Say("Starting VPNKit..."),
					mockVpnKit.EXPECT().Start(),
					mockVpnKit.EXPECT().Watch(localExitChan),
					mockUI.EXPECT().Say("Starting the VM..."),
					mockHypervisor.EXPECT().Start("cfdev"),
					mockUI.EXPECT().Say("Waiting for Garden..."),
					mockProvisioner.EXPECT().Ping(),
					mockUI.EXPECT().Say("Deploying the BOSH Director..."),
					mockProvisioner.EXPECT().DeployBosh(),
					mockUI.EXPECT().Say("Deploying CF..."),
					mockProvisioner.EXPECT().ReportProgress(mockUI, "cf"),
					mockProvisioner.EXPECT().DeployCloudFoundry(nil),
					mockProvisioner.EXPECT().DeployServices(mockUI, []provision.Service{
						{
							Name:       "some-service",
							Handle:     "some-handle",
							Script:     "/path/to/some-script",
							Deployment: "some-deployment",
						},
						{
							Name:       "some-other-service",
							Handle:     "some-other-handle",
							Script:     "/path/to/some-other-script",
							Deployment: "some-other-deployment",
						},
					}),

					mockToggle.EXPECT().Get().Return(false),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_END),
				)

				Expect(startCmd.Execute(start.Args{
					Cpus: 7,
					Mem:  0,
				})).To(Succeed())
			})

			Context("when catalog includes cfdevd", func(){
				BeforeEach(func(){
					startCmd.Config.Dependencies = resource.Catalog{
						Items: []resource.Item{
							{Name: "some-item"},
							{Name: "cfdevd"},
							{Name: "cf-deps.iso"},
						},
					}
				})
				It("downloads cfdevd first", func(){
					if runtime.GOOS == "darwin" {
						mockUI.EXPECT().Say("Installing cfdevd network helper...")
						mockCFDevD.EXPECT().Install()
					}

					gomock.InOrder(
						mockToggle.EXPECT().SetProp("type", "cf"),
						mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
						mockHost.EXPECT().CheckRequirements(),
						mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),
						mockUI.EXPECT().Say("Downloading Network Helper..."),
						mockCache.EXPECT().Sync(resource.Catalog{
							Items: []resource.Item{
								{Name: "cfdevd"},
							},
						}),
						mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
						mockUI.EXPECT().Say("Downloading Resources..."),
						mockCache.EXPECT().Sync(resource.Catalog{
							Items: []resource.Item{
								{Name: "some-item"},
								{Name: "cf-deps.iso"},
							},
						}),
						mockIsoReader.EXPECT().Read(depsIsoPath).Return(metadata, nil),
						mockUI.EXPECT().Say("Creating the VM..."),
						mockHypervisor.EXPECT().CreateVM(hypervisor.VM{
							Name: "cfdev",
							CPUs:     7,
							MemoryMB: 8765,
							DepsIso:  filepath.Join(cacheDir, "cf-deps.iso"),
						}),
						mockUI.EXPECT().Say("Starting VPNKit..."),
						mockVpnKit.EXPECT().Start(),
						mockVpnKit.EXPECT().Watch(localExitChan),
						mockUI.EXPECT().Say("Starting the VM..."),
						mockHypervisor.EXPECT().Start("cfdev"),
						mockUI.EXPECT().Say("Waiting for Garden..."),
						mockProvisioner.EXPECT().Ping(),
						mockUI.EXPECT().Say("Deploying the BOSH Director..."),
						mockProvisioner.EXPECT().DeployBosh(),
						mockUI.EXPECT().Say("Deploying CF..."),
						mockProvisioner.EXPECT().ReportProgress(mockUI, "cf"),
						mockProvisioner.EXPECT().DeployCloudFoundry(nil),
						mockProvisioner.EXPECT().DeployServices(mockUI, []provision.Service{
							{
								Name:       "some-service",
								Handle:     "some-handle",
								Script:     "/path/to/some-script",
								Deployment: "some-deployment",
							},
							{
								Name:       "some-other-service",
								Handle:     "some-other-handle",
								Script:     "/path/to/some-other-script",
								Deployment: "some-other-deployment",
							},
						}),

						mockToggle.EXPECT().Get().Return(true),
						mockAnalyticsD.EXPECT().Start(),
						mockAnalyticsClient.EXPECT().Event(cfanalytics.START_END),
					)

					Expect(startCmd.Execute(start.Args{
						Cpus: 7,
						Mem:  0,
					})).To(Succeed())
				})
			})

			Context("when no args are provided AND deps.iso does not have default memory", func() {
				It("starts the vm with a default memory setting", func() {
					metadata.DefaultMemory = 0

					if runtime.GOOS == "darwin" {
						mockUI.EXPECT().Say("Installing cfdevd network helper...")
						mockCFDevD.EXPECT().Install()
					}

					gomock.InOrder(
						mockToggle.EXPECT().SetProp("type", "cf"),
						mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
						mockHost.EXPECT().CheckRequirements(),
						mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),

						mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
						mockUI.EXPECT().Say("Downloading Resources..."),
						mockCache.EXPECT().Sync(resource.Catalog{
							Items: []resource.Item{
								{Name: "some-item"},
								{Name: "cf-deps.iso"},
							},
						}),
						mockIsoReader.EXPECT().Read(depsIsoPath).Return(metadata, nil),

						mockUI.EXPECT().Say("Creating the VM..."),
						mockHypervisor.EXPECT().CreateVM(hypervisor.VM{
							Name:     "cfdev",
							CPUs:     7,
							MemoryMB: 4192,
							DepsIso:  filepath.Join(cacheDir, "cf-deps.iso"),
						}),
						mockUI.EXPECT().Say("Starting VPNKit..."),
						mockVpnKit.EXPECT().Start(),
						mockVpnKit.EXPECT().Watch(localExitChan),
						mockUI.EXPECT().Say("Starting the VM..."),
						mockHypervisor.EXPECT().Start("cfdev"),
						mockUI.EXPECT().Say("Waiting for Garden..."),
						mockProvisioner.EXPECT().Ping(),
						mockUI.EXPECT().Say("Deploying the BOSH Director..."),
						mockProvisioner.EXPECT().DeployBosh(),
						mockUI.EXPECT().Say("Deploying CF..."),
						mockProvisioner.EXPECT().ReportProgress(mockUI, "cf"),
						mockProvisioner.EXPECT().DeployCloudFoundry(nil),
						mockProvisioner.EXPECT().DeployServices(mockUI, []provision.Service{
							{
								Name:       "some-service",
								Handle:     "some-handle",
								Script:     "/path/to/some-script",
								Deployment: "some-deployment",
							},
							{
								Name:       "some-other-service",
								Handle:     "some-other-handle",
								Script:     "/path/to/some-other-script",
								Deployment: "some-other-deployment",
							},
						}),

						mockToggle.EXPECT().Get().Return(true),
						mockAnalyticsD.EXPECT().Start(),
						mockAnalyticsClient.EXPECT().Event(cfanalytics.START_END),

					)

					Expect(startCmd.Execute(start.Args{
						Cpus: 7,
						Mem:  0,
					})).To(Succeed())
				})
			})
		})

		Context("when the --no-provision flag is provided", func() {
			It("starts the VM and garden but does not provision", func() {
				if runtime.GOOS == "darwin" {
					mockUI.EXPECT().Say("Installing cfdevd network helper...")
					mockCFDevD.EXPECT().Install()
				}

				gomock.InOrder(
					mockToggle.EXPECT().SetProp("type", "cf"),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
					mockHost.EXPECT().CheckRequirements(),
					mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),
					mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
					mockUI.EXPECT().Say("Downloading Resources..."),
					mockCache.EXPECT().Sync(resource.Catalog{
						Items: []resource.Item{
							{Name: "some-item"},
							{Name: "cf-deps.iso"},
						},
					}),
					mockIsoReader.EXPECT().Read(depsIsoPath).Return(metadata, nil),
					mockUI.EXPECT().Say("Creating the VM..."),
					mockHypervisor.EXPECT().CreateVM(hypervisor.VM{
						Name:     "cfdev",
						CPUs:     7,
						MemoryMB: 6666,
						DepsIso:  filepath.Join(cacheDir, "cf-deps.iso"),
					}),
					mockUI.EXPECT().Say("Starting VPNKit..."),
					mockVpnKit.EXPECT().Start(),
					mockVpnKit.EXPECT().Watch(localExitChan),
					mockUI.EXPECT().Say("Starting the VM..."),
					mockHypervisor.EXPECT().Start("cfdev"),
					mockUI.EXPECT().Say("Waiting for Garden..."),
					mockProvisioner.EXPECT().Ping(),
				)

				//no provision message message
				mockUI.EXPECT().Say(gomock.Any())

				Expect(startCmd.Execute(start.Args{
					Cpus:        7,
					Mem:         6666,
					NoProvision: true,
				})).To(Succeed())
			})
		})

		Context("when the -f flag is provided with a non-existing filepath", func() {
			It("returns an error message and does not execute start command", func() {
				Expect(startCmd.Execute(start.Args{
					Cpus:        7,
					Mem:         6666,
					DepsIsoPath: "/wrong-path-to-some-deps.iso",
				}).Error()).To(ContainSubstring("no file found"))
			})
		})

		Context("when the -f flag is provided with an incompatible deps iso version", func() {
			It("returns an error message and does not execute start command", func() {
				customIso := filepath.Join(tmpDir, "custom.iso")
				ioutil.WriteFile(customIso, []byte{}, 0644)
				metadata.Version = "v100"

				if runtime.GOOS == "darwin" {
					mockUI.EXPECT().Say("Installing cfdevd network helper...")
					mockCFDevD.EXPECT().Install()
				}

				gomock.InOrder(
					mockToggle.EXPECT().SetProp("type", "custom.iso"),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
					mockHost.EXPECT().CheckRequirements(),
					mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),
					mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
					mockUI.EXPECT().Say("Downloading Resources..."),
					// don't download cf-deps.iso that we won't use
					mockCache.EXPECT().Sync(resource.Catalog{
						Items: []resource.Item{
							{Name: "some-item"},
						},
					}),
					mockIsoReader.EXPECT().Read(customIso).Return(metadata, nil),
				)

				Expect(startCmd.Execute(start.Args{
					Cpus:        7,
					Mem:         6666,
					DepsIsoPath: customIso,
				})).To(MatchError("custom.iso is not compatible with CF Dev. Please use a compatible file"))
			})
		})

		Context("when the -f flag is provided with an existing filepath", func() {
			It("starts the given iso, doesn't download cf-deps.iso, adds the iso name as an analytics property", func() {
				customIso := filepath.Join(tmpDir, "custom.iso")
				ioutil.WriteFile(customIso, []byte{}, 0644)

				if runtime.GOOS == "darwin" {
					mockUI.EXPECT().Say("Installing cfdevd network helper...")
					mockCFDevD.EXPECT().Install()
				}

				gomock.InOrder(
					mockToggle.EXPECT().SetProp("type", "custom.iso"),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
					mockHost.EXPECT().CheckRequirements(),
					mockHypervisor.EXPECT().IsRunning("cfdev").Return(false, nil),
					mockHostNet.EXPECT().AddLoopbackAliases("some-bosh-director-ip", "some-cf-router-ip"),
					mockUI.EXPECT().Say("Downloading Resources..."),
					// don't download cf-deps.iso that we won't use
					mockCache.EXPECT().Sync(resource.Catalog{
						Items: []resource.Item{
							{Name: "some-item"},
						},
					}),
					mockIsoReader.EXPECT().Read(customIso).Return(metadata, nil),
					mockUI.EXPECT().Say("Creating the VM..."),
					mockHypervisor.EXPECT().CreateVM(hypervisor.VM{
						Name:     "cfdev",
						CPUs:     7,
						MemoryMB: 6666,
						DepsIso:  customIso,
					}),
					mockUI.EXPECT().Say("Starting VPNKit..."),
					mockVpnKit.EXPECT().Start(),
					mockVpnKit.EXPECT().Watch(localExitChan),
					mockUI.EXPECT().Say("Starting the VM..."),
					mockHypervisor.EXPECT().Start("cfdev"),
					mockUI.EXPECT().Say("Waiting for Garden..."),
					mockProvisioner.EXPECT().Ping(),
					mockUI.EXPECT().Say("Deploying the BOSH Director..."),
					mockProvisioner.EXPECT().DeployBosh(),
					mockUI.EXPECT().Say("Deploying CF..."),
					mockProvisioner.EXPECT().ReportProgress(mockUI, "cf"),
					mockProvisioner.EXPECT().DeployCloudFoundry(nil),

					mockProvisioner.EXPECT().DeployServices(mockUI, []provision.Service{
						{
							Name:       "some-service",
							Handle:     "some-handle",
							Script:     "/path/to/some-script",
							Deployment: "some-deployment",
						},
						{
							Name:       "some-other-service",
							Handle:     "some-other-handle",
							Script:     "/path/to/some-other-script",
							Deployment: "some-other-deployment",
						},
					}),

					mockToggle.EXPECT().Get().Return(true),
					mockAnalyticsD.EXPECT().Start(),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_END),
				)

				Expect(startCmd.Execute(start.Args{
					Cpus:        7,
					Mem:         6666,
					DepsIsoPath: customIso,
				})).To(Succeed())
			})
		})

		Context("when linuxkit is already running", func() {
			It("says cf dev is already running", func() {
				gomock.InOrder(
					mockToggle.EXPECT().SetProp("type", "cf"),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_BEGIN),
					mockHost.EXPECT().CheckRequirements(),
					mockHypervisor.EXPECT().IsRunning("cfdev").Return(true, nil),
					mockUI.EXPECT().Say("CF Dev is already running..."),
					mockAnalyticsClient.EXPECT().Event(cfanalytics.START_END, map[string]interface{}{"alreadyrunning": true}),
				)

				Expect(startCmd.Execute(start.Args{})).To(Succeed())
			})
		})
	})
})
