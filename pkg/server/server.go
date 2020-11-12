package server

import (
	"context"
	"crypto/md5"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/yaoice/sample-device-plugin/pkg/utils"
	"google.golang.org/grpc"
	"io/ioutil"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"net"
	"path/filepath"
	"strings"
	"time"
)

// SampleServer是一个device plugin实现
type SampleServer struct {
	srv         *grpc.Server
	devices     map[string]*pluginapi.Device
	notify      chan bool
	ctx         context.Context
	cancel      context.CancelFunc
	restartFlag bool
}

// 初始化SampleServer实例
func NewSampleServer() *SampleServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SampleServer{
		devices:     make(map[string]*pluginapi.Device),
		srv:         grpc.NewServer(grpc.EmptyServerOption{}),
		notify:      make(chan bool),
		ctx:         ctx,
		cancel:      cancel,
		restartFlag: false,
	}
}

func (s *SampleServer) Run() error {
	// 获取本地所有设备
	err := s.listDevice()
	if err != nil {
		log.Fatalf("list device err: %v", err)
	}

	go func() {
		err := s.watchDevice()
		if err != nil {
			log.Println("watch devices error")
		}
	}()

	pluginapi.RegisterDevicePluginServer(s.srv, s)

	//remove sample.socket at first
	sampleSocketPath := filepath.Join(DevicePluginPath, sampleSocket)
	if err = utils.UnlinkFile(sampleSocketPath); err != nil {
		log.Fatalf("unlink %s err: %v", sampleSocket, err)
		return err
	}

	l, err := net.Listen("unix", sampleSocketPath)
	if err != nil {
		log.Fatalf("listen on unix socket %s err: %v", sampleSocket, err)
		return err
	}

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0
		for {
			log.Println("start grpc server for", resourceName)
			err = s.srv.Serve(l)
			if err == nil {
				break
			}

			log.Printf("grpc server for %s crashed with err: %v", resourceName, err)

			if restartCount > 5 {
				log.Fatalln("grpc server for %s crashed after over 5 restart count", resourceName)
			}
			timeSinceLastCrash := time.Since(lastCrashTime).Seconds()
			lastCrashTime = time.Now()
			if timeSinceLastCrash > 3600 {
				restartCount = 1
			} else {
				restartCount++
			}
		}
	}()

	conn, err := s.dial(sampleSocketPath, 5*time.Second)
	if err != nil {
		log.Fatalf("dial %s err: %v", sampleSocket, err)
		return err
	}
	defer conn.Close()
	return nil
}

// GetDevicePluginOptions
func (s *SampleServer) GetDevicePluginOptions(ctx context.Context, e *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	log.Infoln("GetDevicePluginOptions called")
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: true,
	}, nil
}

// ListAndWatch
func (s *SampleServer) ListAndWatch(e *pluginapi.Empty, srv pluginapi.DevicePlugin_ListAndWatchServer) error {
	log.Infoln("ListAndWatch called")
	devs := make([]*pluginapi.Device, len(s.devices))

	i := 0
	for _, dev := range s.devices {
		devs[i] = dev
		i++
	}

	err := srv.Send(&pluginapi.ListAndWatchResponse{
		Devices: devs,
	})
	if err != nil {
		log.Errorf("ListAndWatch send device err: %v", err)
		return err
	}

	// update device list
	for {
		log.Infoln("waiting for device changes")
		select {
		case <-s.notify:
			log.Infoln("update device list, devices:", len(s.devices))
			devs := make([]*pluginapi.Device, len(s.devices))
			i := 0
			for _, dev := range s.devices {
				devs[i] = dev
				i++
			}

			srv.Send(&pluginapi.ListAndWatchResponse{
				Devices: devs,
			})
		case <-s.ctx.Done():
			log.Infoln("ListAndWatch exit")
			return nil
		}
	}
}

// Allocate
func (s *SampleServer) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.Infoln("Allocate called")
	resps := &pluginapi.AllocateResponse{}

	for _, req := range reqs.ContainerRequests {
		log.Infof("received request: %v", strings.Join(req.DevicesIDs, ","))
		resp := &pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				"SAMPLE_DEVICES": strings.Join(req.DevicesIDs, ","),
			},
		}
		resps.ContainerResponses = append(resps.ContainerResponses, resp)
	}
	return resps, nil
}

// PreStartContainer
func (s *SampleServer) PreStartContainer(ctx context.Context, req *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	log.Infoln("PreStartContainer called")
	return &pluginapi.PreStartContainerResponse{}, nil
}

// GetPreferredAllocation
func (s *SampleServer) GetPreferredAllocation(ctx context.Context, req *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	log.Infoln("GetPreferredAllocation called")
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// 向kubelet注册device plugin
func (s *SampleServer) RegisterToKubelet() error {
	socketFile := filepath.Join(DevicePluginPath, KubeletSocket)

	conn, err := s.dial(socketFile, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	req := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     sampleSocket,
		ResourceName: resourceName,
	}
	log.Infof("Register to kubelet with endpoint %s", req.Endpoint)
	_, err = client.Register(context.Background(), req)
	if err != nil {
		return err
	}
	return nil
}

func (s *SampleServer) dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			var timeOut time.Duration
			if deadline, ok := ctx.Deadline(); ok {
				timeOut = time.Until(deadline)
			} else {
				timeOut = 0
			}
			return net.DialTimeout("unix", addr, timeOut)
		}))
	if err != nil {
		return nil, err
	}
	return c, nil
}

// 节点上获取所有设备
func (s *SampleServer) listDevice() error {
	dir, err := ioutil.ReadDir(defaultSampleLocation)
	if err != nil {
		return err
	}

	for _, f := range dir {
		if f.IsDir() {
			continue
		}

		md5Sum := md5.Sum([]byte(f.Name()))
		s.devices[f.Name()] = &pluginapi.Device{
			ID:     string(md5Sum[:]),
			Health: pluginapi.Healthy,
		}
		log.Infof("find device %s", f.Name())
	}
	return nil
}

// 节点上监控设备
func (s *SampleServer) watchDevice() error {
	log.Infoln("watching devices")
	// 初始化一个watcher对象
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer func() {
			log.Infoln("watch device exit!")
		}()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					continue
				}
				log.Infoln("the event of device:", event.Op.String())

				// Create事件
				if event.Op&fsnotify.Create == fsnotify.Create {
					sum := md5.Sum([]byte(event.Name))
					s.devices[event.Name] = &pluginapi.Device{
						ID:     string(sum[:]),
						Health: pluginapi.Healthy,
					}
					s.notify <- true
					// Remove事件
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					delete(s.devices, event.Name)
					s.notify <- true
					log.Infoln("device deleted:", event.Name)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("receive error:", err)
				}
			case <-s.ctx.Done():
				break
			}
		}
	}()

	err = watcher.Add(defaultSampleLocation)
	if err != nil {
		return err
	}
	<-done

	return nil
}
