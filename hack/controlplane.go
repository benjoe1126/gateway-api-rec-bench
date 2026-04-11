package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kc "k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func main() {
	env := envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("../", "crds")},
		BinaryAssetsDirectory: filepath.Join("../bin"),
		DownloadBinaryAssets:  true,
	}
	cfg, err := env.Start()
	log.Println(cfg, err)
	if err != nil {
		log.Fatal(err)
	}
	kubeconfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"envtest": &clientcmdapi.Cluster{
				Server:                   cfg.Host,
				CertificateAuthorityData: cfg.CAData,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"envtest": &clientcmdapi.AuthInfo{
				ClientCertificateData: cfg.CertData,
				ClientKeyData:         cfg.KeyData,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"envtest": &clientcmdapi.Context{
				Cluster:  "envtest",
				AuthInfo: "envtest",
			},
		},
		CurrentContext: "envtest",
	}
	client, err := kc.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	gatewayNs := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "envoy-gateway-system",
		},
		Spec: apiv1.NamespaceSpec{},
	}
	client.CoreV1().Namespaces().Create(context.Background(), gatewayNs, metav1.CreateOptions{})
	if err := clientcmd.WriteToFile(kubeconfig, "../tmp/kubeconfig.yaml"); err != nil {
		log.Fatal(err)
	}
	if err != nil {
		panic(err)
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	stopChan := make(chan bool, 1)
	go func() {
		<-sigc
		stopChan <- true
		return
	}()
	for {
		select {
		case <-stopChan:
			return
		default:
		}
	}
	if err := env.Stop(); err != nil {
		panic(err)
	}
}
