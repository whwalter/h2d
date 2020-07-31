package main

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"

	utils "github.com/maorfr/helm-plugin-utils/pkg"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var clientset kubernetes.Interface

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if err := newCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm2culler",
		Short: "Remove tiller if there are no v2 releases in a namespace.",
		RunE:  detectorRunE,
	}

	if clientset == nil {
		clientset = utils.GetClientSet()
	}

	cmd.PersistentFlags().Bool("remove-tiller", false, "remove tiller from empty namespaces")
	cmd.PersistentFlags().Bool("remove-service-account", false, "remove tiller serviceaccount from empty namespaces")
	cmd.PersistentFlags().String("service-account", "", "tiller serviceaccount name, required for serviceaccount removal")
	cmd.PersistentFlags().String("label-selector", "name=tiller,app=helm", "tiller labels")
	return cmd
}

func detectorRunE(cmd *cobra.Command, args []string) error {

	label, err := cmd.Flags().GetString("label-selector")
	if err != nil {
		return err
	}
	if label == "" || len(strings.Split(label, "=")) == 1 {
		return fmt.Errorf("Invalid label selector for tiller: %s", label)
	}

	deleteTiller, err := cmd.Flags().GetBool("remove-tiller")
	if err != nil {
		return err
	}

	deleteTillerSA, err := cmd.Flags().GetBool("remove-service-account")
	if err != nil {
		return err
	}

	tillerSA, err := cmd.Flags().GetString("service-account")
	if err != nil {
		return err
	}

	if (deleteTillerSA && (tillerSA == "")) || (!deleteTillerSA && (tillerSA != "")) {
		return errors.New("remove-service-accounts and service-account requried to remove service accounts")
	}

	return processNamespaces(clientset, label, tillerSA, deleteTiller, deleteTillerSA)
}

func processNamespaces(cs kubernetes.Interface, label, tillerSA string, deleteTiller, deleteTillerSA bool) error {

	//Get namespaces that have a tiller in them
	opts := metav1.ListOptions{LabelSelector: label}
	tillerDeployments, err := clientset.AppsV1().Deployments("").List(opts)
	if err != nil {
		return err
	}
	namespaces := []string{}

	for _, tiller := range tillerDeployments.Items {
		namespaces = append(namespaces, tiller.ObjectMeta.Namespace)
	}

	removalNamespaces := []string{}
	errs := map[string]error{}

	//Look for configMap backed releases in these namespaces
	for _, namespace := range namespaces {
		releases, err := getTillerReleases(namespace)
		if err != nil {
			errs[namespace] = err
		}
		if len(releases) > 0 {
			log.WithFields(log.Fields{
				"namespace": namespace,
				"releases":  releases,
			}).Info("Helm2 releases detected")
		} else {
			removalNamespaces = append(removalNamespaces, namespace)
		}
	}

	//If a namespace has no configMap backed releases and a tiller, remove tiller
	for _, ns := range removalNamespaces {
		nsLogger := log.WithFields(log.Fields{
			"namespace": ns,
		})

		if !deleteTiller {
			nsLogger.Info("DRYRUN: Delete tiller has been disabled")
		}
		nsLogger.Info("Removing tiller resources")
		if deleteTiller {
			err := removeTiller(ns, label, tillerSA, deleteTillerSA)
			if err != nil {
				nsLogger.Error(err)
			}
		}
	}
	if len(errs) > 0 {
		report, err := json.MarshalIndent(errs, "", "    ")
		if err != nil {
			return err
		}
		return errors.New(string(report))
	}
	return nil
}

func getTillerReleases(namespace string) ([]string, error) {
	var releaseList []string
	releases, err := utils.ListReleaseNamesInNamespace(utils.ListReleaseNamesInNamespaceOptions{TillerNamespace: namespace, Namespace: namespace})
	if err != nil {
		return []string{}, err
	}
	if releases != "" {
		releaseList = strings.Split(releases, ",")
	} else {
		releaseList = []string{}
	}

	return releaseList, nil
}

func removeTiller(namespace, label, tillerSA string, deleteTillerSA bool) error {

	nsLogger := log.WithFields(log.Fields{
		"namespace": namespace,
	})

	var errs []error
	opts := metav1.ListOptions{LabelSelector: label}
	appsv1 := clientset.AppsV1()
	corev1 := clientset.CoreV1()

	tillerDeployments, err := appsv1.Deployments(namespace).List(opts)
	if err != nil {
		errs = append(errs, err)
	}

	tillerSvcs, err := corev1.Services(namespace).List(opts)
	if err != nil {
		errs = append(errs, err)
	}

	for _, tiller := range tillerDeployments.Items {
		nsLogger.Info(fmt.Sprintf("Found tiller deployment. Deleting tiller deployment %s\n", tiller.ObjectMeta.Name))
		err = appsv1.Deployments(namespace).Delete(tiller.ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, tiller := range tillerSvcs.Items {
		nsLogger.Info(fmt.Sprintf("Found tiller service. Deleting tiller service %s\n", tiller.ObjectMeta.Name))
		err = corev1.Services(namespace).Delete(tiller.ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	if deleteTillerSA {
		serviceAccounts, err := corev1.ServiceAccounts(namespace).List(opts)
		if err != nil {
			errs = append(errs, err)
		}
		for _, sa := range serviceAccounts.Items {
			nsLogger.Info(fmt.Sprintf("Found tiller service account. Deleteing tiller service account %s\n", sa.ObjectMeta.Name))
			err = corev1.ServiceAccounts(namespace).Delete(sa.ObjectMeta.Name, &metav1.DeleteOptions{})
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		eJSON, _ := json.Marshal(errs)
		return errors.New(string(eJSON))
	}
	return nil
}
