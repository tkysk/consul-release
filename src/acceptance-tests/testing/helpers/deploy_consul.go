package helpers

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/pbkdf2"

	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/consul"
	"acceptance-tests/testing/destiny"
)

func DeployConsulWithInstanceCount(count int, client bosh.Client, config Config) (manifest destiny.Manifest, kv consul.KV, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := destiny.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("consul-%s", guid),
	}

	switch info.CPI {
	case "aws_cpi":
		manifestConfig.IAAS = destiny.AWS
		if config.AWS.Subnet != "" {
			manifestConfig.AWS.Subnet = config.AWS.Subnet
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}
	case "warden_cpi":
		manifestConfig.IAAS = destiny.Warden
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest = destiny.NewConsul(manifestConfig)

	manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, count)

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	manifest, err = destiny.FromYAML(yaml)
	if err != nil {
		return
	}

	err = client.Deploy(yaml)
	if err != nil {
		return
	}

	kv, err = NewKV(manifest, count)

	return
}

func NewKV(manifest destiny.Manifest, count int) (kv consul.KV, err error) {
	members := manifest.ConsulMembers()
	if len(members) != count {
		err = fmt.Errorf("expected %d consul members, found %d", count, len(members))
		return
	}

	consulMemberAddresses := []string{}
	for _, member := range members {
		consulMemberAddresses = append(consulMemberAddresses, member.Address)
	}

	dataDir, err := ioutil.TempDir("", "consul")
	if err != nil {
		return
	}

	configDir, err := ioutil.TempDir("", "consul-config")
	if err != nil {
		return
	}

	var encryptKey string
	if len(manifest.Properties.Consul.EncryptKeys) > 0 {
		key := manifest.Properties.Consul.EncryptKeys[0]
		encryptKey = base64.StdEncoding.EncodeToString(pbkdf2.Key([]byte(key), []byte(""), 20000, 16, sha1.New))
	}

	agent := consul.NewAgent(consul.AgentOptions{
		DataDir:    dataDir,
		RetryJoin:  consulMemberAddresses,
		ConfigDir:  configDir,
		Domain:     "cf.internal",
		Key:        manifest.Properties.Consul.AgentKey,
		Cert:       manifest.Properties.Consul.AgentCert,
		CACert:     manifest.Properties.Consul.CACert,
		Encrypt:    encryptKey,
		ServerName: "consul agent",
	})

	agentLocation := "http://127.0.0.1:8500"

	kv = consul.NewManagedKV(consul.ManagedKVConfig{
		Agent:   agent,
		KV:      consul.NewHTTPKV(agentLocation),
		Catalog: consul.NewHTTPCatalog(agentLocation),
	})

	return
}
