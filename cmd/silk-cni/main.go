package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/validator.v2"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
	"github.com/vishvananda/netlink"

	"code.cloudfoundry.org/silk/cni/adapter"
	"code.cloudfoundry.org/silk/cni/config"
	"code.cloudfoundry.org/silk/cni/lib"
	"code.cloudfoundry.org/silk/cni/netinfo"
	"code.cloudfoundry.org/silk/daemon"
	libAdapter "code.cloudfoundry.org/silk/lib/adapter"
	"code.cloudfoundry.org/silk/lib/datastore"
	"code.cloudfoundry.org/silk/lib/filelock"
	"code.cloudfoundry.org/silk/lib/serial"
	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
)

type CNIPlugin struct {
	HostNSPath      string
	HostNS          ns.NetNS
	ConfigCreator   *config.ConfigCreator
	VethPairCreator *lib.VethPairCreator
	Host            *lib.Host
	Container       *lib.Container
	Store           *datastore.Store
	Logger          lager.Logger
}

func main() {
	hostNS, err := ns.GetCurrentNS()
	if err != nil {
		log.Fatal(err)
	}

	logger := lager.NewLogger("silk-cni")
	sink := lager.NewWriterSink(os.Stderr, lager.INFO)
	logger.RegisterSink(sink)

	netlinkAdapter := &libAdapter.NetlinkAdapter{}
	linkOperations := &lib.LinkOperations{
		SysctlAdapter:  &adapter.SysctlAdapter{},
		NetlinkAdapter: netlinkAdapter,
		Logger:         logger,
	}
	commonSetup := &lib.Common{
		NetlinkAdapter: netlinkAdapter,
		LinkOperations: linkOperations,
	}
	store := &datastore.Store{
		Serializer: &serial.Serial{},
		Locker:     &filelock.Locker{},
	}

	plugin := &CNIPlugin{
		HostNSPath: hostNS.Path(),
		HostNS:     hostNS,
		ConfigCreator: &config.ConfigCreator{
			HardwareAddressGenerator: &config.HardwareAddressGenerator{},
			DeviceNameGenerator:      &config.DeviceNameGenerator{},
			NamespaceAdapter:         &adapter.NamespaceAdapter{},
		},
		VethPairCreator: &lib.VethPairCreator{
			NetlinkAdapter: netlinkAdapter,
		},
		Host: &lib.Host{
			Common:         commonSetup,
			LinkOperations: linkOperations,
		},
		Container: &lib.Container{
			Common:         commonSetup,
			LinkOperations: linkOperations,
		},
		Logger: logger,
		Store:  store,
	}

	skel.PluginMain(plugin.cmdAdd, plugin.cmdDel, version.PluginSupports("0.3.1"))
}

type NetConf struct {
	types.NetConf
	DataDir         string          `json:"dataDir"`
	SubnetFile      string          `json:"subnetFile"`
	MTU             int             `json:"mtu" validate:"min=0"`
	Datastore       string          `json:"datastore"`
	DaemonPort      int             `json:"daemonPort"`
	BandwidthLimits BandwidthLimits `json:"bandwidthLimits"`
}

type BandwidthLimits struct {
	Rate  int `json:"rate"`
	Burst int `json:"burst"`
}

type HostLocalIPAM struct {
	Type    string         `json:"type"`
	Subnet  string         `json:"subnet"`
	Gateway string         `json:"gateway"`
	Routes  []*types.Route `json:"routes"`
	DataDir string         `json:"dataDir"`
}

type ConfForHostLocal struct {
	CNIVersion string        `json:"cniVersion"`
	Name       string        `json:"name"`
	IPAM       HostLocalIPAM `json:"ipam"`
}

func typedError(msg string, err error) *types.Error {
	return &types.Error{
		Code:    100,
		Msg:     msg,
		Details: err.Error(),
	}
}

func getNetworkInfo(netConf NetConf) (daemon.NetworkInfo, error) {
	err := validator.Validate(netConf)
	if err != nil {
		return daemon.NetworkInfo{}, fmt.Errorf("invalid config: %s", err)
	}

	discoverer := netinfo.Discoverer{}
	if netConf.SubnetFile != "" {
		discoverer.NetInfo = &netinfo.Flannel{
			SubnetFilePath: netConf.SubnetFile,
		}
	} else {
		jsonClient := json_client.New(lager.NewLogger(""), http.DefaultClient, fmt.Sprintf("http://127.0.0.1:%d", netConf.DaemonPort))
		discoverer.NetInfo = &netinfo.Daemon{
			JSONClient: jsonClient,
		}
	}
	return discoverer.Discover(netConf.MTU)
}

func (p *CNIPlugin) cmdAdd(args *skel.CmdArgs) error {
	var netConf NetConf
	err := json.Unmarshal(args.StdinData, &netConf)
	if err != nil {
		return err // impossible, skel package asserts JSON is valid
	}

	networkInfo, err := getNetworkInfo(netConf)
	if err != nil {
		return typedError("discover network info", err)
	}

	generator := config.IPAMConfigGenerator{}
	ipamConfig := generator.GenerateConfig(networkInfo.OverlaySubnet, netConf.Name, netConf.DataDir)
	ipamConfigBytes, _ := json.Marshal(ipamConfig) // untestable

	result, err := invoke.DelegateAdd("host-local", ipamConfigBytes)
	if err != nil {
		return typedError("run ipam plugin", err)
	}

	cniResult, err := current.NewResultFromResult(result)
	if err != nil {
		return fmt.Errorf("convert result to current CNI version: %s", err) // not tested
	}

	cfg, err := p.ConfigCreator.Create(p.HostNS, args, cniResult, networkInfo.MTU)
	if err != nil {
		return typedError("create config", err)
	}

	err = p.VethPairCreator.Create(cfg)
	if err != nil {
		return typedError("create veth pair", err)
	}

	err = p.Host.Setup(cfg)
	if err != nil {
		return typedError("set up host", err)
	}

	// err = exec.Command("tc", "qdisc", "add", "dev", cfg.Host.DeviceName, "root", "tbf",
	// 	"rate", rateString, "burst", burstString, "latency", "100ms").Run()
	if netConf.BandwidthLimits.Rate > 0 && netConf.BandwidthLimits.Burst > 0 {
		link, err := netlink.LinkByName(cfg.Host.DeviceName)
		if err != nil {
			panic(err)
			return typedError("BANANA", err)
		}
		rateInBits := netConf.BandwidthLimits.Rate
		rateInBytes := rateInBits / 8
		burstInBits := netConf.BandwidthLimits.Burst
		bufferInBytes := burstInBits * 1000000000 / rateInBits / 8
		limitInBytes := rateInBytes / 10

		qdisc := &netlink.Tbf{
			QdiscAttrs: netlink.QdiscAttrs{
				LinkIndex: link.Attrs().Index,
				Handle:    netlink.MakeHandle(1, 0),
				Parent:    netlink.HANDLE_ROOT,
			},
			Limit:  uint32(limitInBytes / 10),
			Rate:   uint64(rateInBytes),
			Buffer: uint32(bufferInBytes),
		}
		err = netlink.QdiscAdd(qdisc)
		if err != nil {
			return typedError("throttle container ingress", err)
		}
	}

	err = p.Container.Setup(cfg)
	if err != nil {
		return typedError("set up container", err)
	}

	// use args.Netns as the 'handle' for now
	err = p.Store.Add(netConf.Datastore, filepath.Base(args.Netns), cfg.Container.Address.IP.String(), nil)
	if err != nil {
		return typedError("write container metadata", err)
	}

	return types.PrintResult(cfg.AsCNIResult(), netConf.CNIVersion)
}

func (p *CNIPlugin) cmdDel(args *skel.CmdArgs) error {
	var netConf NetConf
	err := json.Unmarshal(args.StdinData, &netConf)
	if err != nil {
		return err // impossible, skel package asserts JSON is valid
	}

	networkInfo, err := getNetworkInfo(netConf)
	if err != nil {
		return typedError("discover network info", err)
	}

	generator := config.IPAMConfigGenerator{}
	ipamConfig := generator.GenerateConfig(networkInfo.OverlaySubnet, netConf.Name, netConf.DataDir)
	ipamConfigBytes, _ := json.Marshal(ipamConfig) // untestable

	err = invoke.DelegateDel("host-local", ipamConfigBytes)
	if err != nil {
		p.Logger.Error("host-local-ipam", err)
		// continue, keep trying to cleanup
	}

	containerNS, err := ns.GetNS(args.Netns)
	if err != nil {
		p.Logger.Error("open-netns", err)
		return nil // can't do teardown if no netns
	}

	err = p.Container.Teardown(containerNS, args.IfName)
	if err != nil {
		return typedError("teardown failed", err)
	}

	_, err = p.Store.Delete(netConf.Datastore, filepath.Base(args.Netns))
	if err != nil {
		p.Logger.Error("write-container-metadata", err)
	}

	return nil
}
