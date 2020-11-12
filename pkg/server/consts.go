package server

const (
	resourceName          string = "sample.com/sample"
	defaultSampleLocation string = "/etc/samples"
	sampleSocket          string = "sample.sock"
	// kubelet监听的unix socket名称
	KubeletSocket string = "kubelet.sock"
	// device plugin放置的默认路径
	DevicePluginPath string = "/var/lib/kubelet/device-plugins/"
)
