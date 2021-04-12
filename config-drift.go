package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Create a new type for a list of Strings
type stringList []string

// Implement the flag.Value interface
func (s *stringList) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringList) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func main() {

	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)

	// Authenticate
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	replicaDriftCommand := flag.NewFlagSet("replicaDrift", flag.ExitOnError)

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand
	if len(os.Args) < 2 {
		fmt.Println("deployment name is required")
		os.Exit(1)
	}

	deploymentTextPtr := replicaDriftCommand.String("deployment", "", "Deployment name. (Required)")

	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	// FlagSet.Parse() requires a set of arguments to parse as input
	// os.Args[2:] will be all arguments starting after the subcommand at os.Args[1]
	switch os.Args[1] {
	case "replicaDrift":
		replicaDriftCommand.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check which subcommand was parsed
	if replicaDriftCommand.Parsed() {
		fmt.Print("\n")
		fmt.Printf("-------------------- deployment: %s -------------------\n", *deploymentTextPtr)
		fmt.Print("\n")

		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		for {
			deploymentName := *deploymentTextPtr
			deploymentInstances := getDeploymentAllNamespaces(clientset, deploymentName)

			m := make(map[string]int32)
			for _, deployment := range deploymentInstances {
				m[deployment.Namespace] = deployment.Status.Replicas
			}

			for k, v := range m {
				fmt.Printf("Namespace: %s - Replicas: %d \n", k, v)
			}
			fmt.Print("--------------------\n")

			time.Sleep(10 * time.Second)
		}
	}

}

func getDeployment(clientset *kubernetes.Clientset, deploymentName string, namespace string) (*appsv1.Deployment, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func getDeploymentAllNamespaces(clientset *kubernetes.Clientset, deploymentName string) []appsv1.Deployment {
	ns := getClusterNamespaces(clientset)
	var deployments []appsv1.Deployment

	for _, namespace := range ns {

		deployment, err := getDeployment(clientset, deploymentName, namespace)

		if errors.IsNotFound(err) {
			log.Infof("not found in namespace %s", namespace)
		} else {
			deployments = append(deployments, *deployment)
		}

	}
	return deployments
}

func getConfigMapAllNamespaces(clientset *kubernetes.Clientset, configMapName string) []*v1.ConfigMap {
	ns := getClusterNamespaces(clientset)
	var configMaps []*v1.ConfigMap
	for index, value := range ns {
		cm, _ := clientset.CoreV1().ConfigMaps(value).Get(context.TODO(), configMapName, metav1.GetOptions{})
		configMaps[index] = cm
	}
	return configMaps
}

// Get all namespaces in the cluster
func getClusterNamespaces(clientset *kubernetes.Clientset) []string {
	namespaces, _ := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	var c []string
	for _, value := range namespaces.Items {
		c = append(c, value.Name)
	}
	return c
}

// Given a Kubernetes connection, get all configmap names for provided namespace
func getConfigMapNames(clientset *kubernetes.Clientset, namespace string) []string {
	configmaps, _ := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	var c []string
	for _, value := range configmaps.Items {
		c = append(c, value.Name)
	}
	return c
}
