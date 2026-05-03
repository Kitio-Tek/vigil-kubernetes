/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Command kubectl-athos is a kubectl plugin for interacting with
// Athos-managed PostgreSQL clusters.
//
// Install:
//
//	go install github.com/Kitio-Tek/athos-kubernetes/cmd/plugin@latest
//	mv $(go env GOPATH)/bin/plugin $(go env GOPATH)/bin/kubectl-athos
//
// Usage:
//
//	kubectl athos status <cluster-name> [-n namespace]
//	kubectl athos failover <cluster-name> --to <pod-name> [-n namespace]
//	kubectl athos backup <cluster-name> [-n namespace]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Kitio-Tek/athos-kubernetes/internal/version"
)

var (
	namespace  string
	kubeconfig string
)

func init() {
	flag.StringVar(&namespace, "n", "default", "Kubernetes namespace")
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	flag.StringVar(&kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file")
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	cmd := args[0]
	switch cmd {
	case "status":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: kubectl athos status <cluster-name>")
			os.Exit(1)
		}
		if err := runStatus(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		runVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runStatus(clusterName string) error {
	dynClient, err := newDynamicClient()
	if err != nil {
		return fmt.Errorf("building Kubernetes client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "pg.athos.io",
		Version:  "v1alpha1",
		Resource: "postgresclusters",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	obj, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting cluster %q in namespace %q: %w", clusterName, namespace, err)
	}

	spec := nestedMap(obj.Object, "spec")
	status := nestedMap(obj.Object, "status")

	phase := nestedString(status, "phase")
	primary := nestedString(status, "currentPrimary")
	ready := nestedInt64(status, "readyInstances")
	instances := nestedInt64(spec, "instances")
	pgVersion := nestedInt64(spec, "postgresVersion")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "NAME\t%s\n", clusterName)
	_, _ = fmt.Fprintf(w, "NAMESPACE\t%s\n", namespace)
	_, _ = fmt.Fprintf(w, "PHASE\t%s\n", phase)
	_, _ = fmt.Fprintf(w, "PRIMARY\t%s\n", primary)
	_, _ = fmt.Fprintf(w, "READY\t%d/%d\n", ready, instances)
	_, _ = fmt.Fprintf(w, "POSTGRES\t%d\n", pgVersion)
	_ = w.Flush()

	return nil
}

func runVersion() {
	info := version.Info()
	fmt.Printf("kubectl-athos %s (%s, built %s)\n", info.Version, info.Commit, info.BuildDate)
	fmt.Printf("  Go:        %s\n", info.GoVersion)
	fmt.Printf("  Platform:  %s\n", info.Platform)
	fmt.Printf("  Operator:  Athos Kubernetes\n")
	fmt.Printf("  API:       pg.athos.io/v1alpha1\n")
}

func printUsage() {
	fmt.Println("kubectl-athos - kubectl plugin for Athos Kubernetes operator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  kubectl athos [flags] <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  status <cluster>   Show cluster status and instance counts")
	fmt.Println("  version            Print plugin and operator version")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -n, --namespace    Kubernetes namespace (default: default)")
	fmt.Println("      --kubeconfig   Path to kubeconfig (default: $KUBECONFIG)")
}

func newDynamicClient() (dynamic.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	return dynamic.NewForConfig(config)
}

func nestedMap(obj map[string]interface{}, field string) map[string]interface{} {
	if obj == nil {
		return nil
	}
	v, ok := obj[field]
	if !ok {
		return nil
	}
	m, _ := v.(map[string]interface{})
	return m
}

func nestedString(obj map[string]interface{}, field string) string {
	if obj == nil {
		return ""
	}
	v, ok := obj[field]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func nestedInt64(obj map[string]interface{}, field string) int64 {
	if obj == nil {
		return 0
	}
	v, ok := obj[field]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int32:
		return int64(n)
	}
	return 0
}
