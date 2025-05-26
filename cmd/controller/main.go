package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/altinity/node-label-controller/pkg/controller"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeconfig string
	var contextName string
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&contextName, "context", "", "name of the kubeconfig context to use")
	flag.Parse()

	config, err := buildConfig(kubeconfig, contextName)
	if err != nil {
		slog.Error("Error building kubeconfig", "error", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Error building kubernetes clientset", "error", err)
		os.Exit(1)
	}

	ctrl := controller.NewController(clientset)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-stopCh
		slog.Info("Received termination signal, shutting down...")
		cancel()
	}()

	slog.Info("Starting node label controller")
	if err := ctrl.Run(ctx, 2); err != nil {
		slog.Error("Error running controller", "error", err)
		os.Exit(1)
	}
}

func buildConfig(kubeconfig string, contextName string) (*rest.Config, error) {
	if kubeconfig != "" {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		configOverrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	}
	return rest.InClusterConfig()
}
