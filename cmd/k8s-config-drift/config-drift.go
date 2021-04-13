package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
		log.Errorf("Deployment name is required")
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

	ns, err := getClusterNamespaces(clientset)
	// If we can't get namespaces from cluster, there's no reason to continue
	if err != nil {
		log.Fatalf("%v", err.Error())
		panic(err.Error())
	}

	var deployments []appsv1.Deployment

	for _, namespace := range ns {

		deployment, err := getDeployment(clientset, deploymentName, namespace)
		if errors.IsNotFound(err) {
			log.Infof("Deployment %s not found in namespace %s", deployment, namespace)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			log.Errorf("Error getting deployment %s in namespace %s: %v \n ",
				deploymentName, namespace, statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			log.Infof("Found deployment %s in namespace %s \n", deploymentName, namespace)
			deployments = append(deployments, *deployment)
		}

	}
	return deployments
}

func getConfigMapAllNamespaces(clientset *kubernetes.Clientset, configMapName string) []*v1.ConfigMap {

	ns, err := getClusterNamespaces(clientset)
	if err != nil {
		log.Fatalf("%s", err.Error())
		panic(err.Error())
	}

	var configMaps []*v1.ConfigMap
	for index, namespace := range ns {

		cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			log.Infof("Configmap %s not found in namespace %s", configMapName, namespace)
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			// While we log the error, we don't halt. This is because, it's likely that the user did not have permissions for that
			// particular namespace. TODO - Add ability to exclude certain namespaces
			log.Errorf("Error getting deployment %s in namespace %s: %v \n ",
				configMapName, namespace, statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			log.Infof("Found configmap %s in namespace %s \n", configMapName, namespace)
			configMaps[index] = cm
		}

	}
	return configMaps
}

// Get all namespaces in the cluster
func getClusterNamespaces(clientset *kubernetes.Clientset) ([]string, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	var c []string
	for _, value := range namespaces.Items {
		c = append(c, value.Name)
	}

	if err != nil {
		return nil, err
	}
	return c, nil
}

// Given a Kubernetes connection, get all configmap names for provided namespace
func getConfigMapNames(clientset *kubernetes.Clientset, namespace string) []string {
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var c []string
	for _, value := range configmaps.Items {
		c = append(c, value.Name)
	}
	return c
}
