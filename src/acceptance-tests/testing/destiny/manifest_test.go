package destiny_test

import (
	"acceptance-tests/testing/destiny"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Manifest", func() {
	Describe("ToYAML", func() {
		It("returns a YAML representation of the consul manifest", func() {
			consulManifest, err := ioutil.ReadFile("fixtures/consul_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest := destiny.NewConsul(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "consul",
				IAAS:         destiny.Warden,
			})

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml).To(MatchYAML(consulManifest))
		})

		It("returns a YAML representation of the turbulence manifest", func() {
			turbulenceManifest, err := ioutil.ReadFile("fixtures/turbulence_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest := destiny.NewTurbulence(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "turbulence",
				IAAS:         destiny.Warden,
				BOSH: destiny.ConfigBOSH{
					Target:   "some-bosh-target",
					Username: "some-bosh-username",
					Password: "some-bosh-password",
				},
			})

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml).To(MatchYAML(turbulenceManifest))
		})
	})

	Describe("FromYAML", func() {
		It("returns a Manifest matching the given YAML", func() {
			consulManifest, err := ioutil.ReadFile("fixtures/consul_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest, err := destiny.FromYAML(consulManifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "consul",
				Releases: []destiny.Release{{
					Name:    "consul",
					Version: "latest",
				}},
				Compilation: destiny.Compilation{
					Network:             "consul1",
					ReuseCompilationVMs: true,
					Workers:             3,
				},
				Update: destiny.Update{
					Canaries:        1,
					CanaryWatchTime: "1000-180000",
					MaxInFlight:     50,
					Serial:          true,
					UpdateWatchTime: "1000-180000",
				},
				ResourcePools: []destiny.ResourcePool{
					{
						Name:    "consul_z1",
						Network: "consul1",
						Stemcell: destiny.ResourcePoolStemcell{
							Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
							Version: "latest",
						},
					},
				},
				Jobs: []destiny.Job{
					{
						Name:      "consul_z1",
						Instances: 1,
						Networks: []destiny.JobNetwork{{
							Name:      "consul1",
							StaticIPs: []string{"10.244.4.4"},
						}},
						PersistentDisk: 1024,
						Properties: &destiny.JobProperties{
							Consul: destiny.JobPropertiesConsul{
								Agent: destiny.JobPropertiesConsulAgent{
									Mode: "server",
									Services: destiny.JobPropertiesConsulAgentServices{
										"router": destiny.JobPropertiesConsulAgentService{
											Name: "gorouter",
											Check: &destiny.JobPropertiesConsulAgentServiceCheck{
												Name:     "router-check",
												Script:   "/var/vcap/jobs/router/bin/script",
												Interval: "1m",
											},
											Tags: []string{"routing"},
										},
										"cloud_controller": destiny.JobPropertiesConsulAgentService{},
									},
								},
							},
						},
						ResourcePool: "consul_z1",
						Templates: []destiny.JobTemplate{{
							Name:    "consul_agent",
							Release: "consul",
						}},
						Update: &destiny.JobUpdate{
							MaxInFlight: 1,
						},
					},
				},
				Networks: []destiny.Network{
					{
						Name: "consul1",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{Name: "random"},
								Gateway:         "10.244.4.1",
								Range:           "10.244.4.0/24",
								Reserved: []string{
									"10.244.4.2-10.244.4.3",
									"10.244.4.12-10.244.4.254",
								},
								Static: []string{
									"10.244.4.4",
									"10.244.4.5",
									"10.244.4.6",
									"10.244.4.7",
									"10.244.4.8",
								},
							},
						},
						Type: "manual",
					},
				},
				Properties: destiny.Properties{
					Consul: &destiny.PropertiesConsul{
						Agent: destiny.PropertiesConsulAgent{
							LogLevel: "",
							Servers: destiny.PropertiesConsulAgentServers{
								Lan: []string{"10.244.4.4"},
							},
						},
						CACert:      destiny.CACert,
						AgentCert:   destiny.AgentCert,
						AgentKey:    destiny.AgentKey,
						ServerCert:  destiny.ServerCert,
						ServerKey:   destiny.ServerKey,
						EncryptKeys: []string{destiny.EncryptKey},
						RequireSSL:  true,
					},
				},
			}))
		})

		Context("failure cases", func() {
			It("should error on malformed YAML", func() {
				_, err := destiny.FromYAML([]byte("%%%%%%%%%%"))
				Expect(err).To(MatchError(ContainSubstring("yaml: ")))
			})
		})
	})
})
