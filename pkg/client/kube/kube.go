package kube

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path/filepath"
	sigyaml "sigs.k8s.io/yaml"
)

type Client struct {
	dynamic dynamic.Interface
	// restClient rest.Interface
	restConfig *rest.Config
	restMapper meta.RESTMapper
	clientSet  *kubernetes.Clientset
	apiConfig  clientcmdapi.Config
	log        *logrus.Logger
}

func New(masterURL string, kubeConfigData []byte, log *logrus.Logger) (*Client, error) {
	apiConfig, err := clientcmd.Load(kubeConfigData)
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromKubeconfigGetter(masterURL, func() (*clientcmdapi.Config, error) {
		return apiConfig, nil
	})
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	log.Printf("==================")
	return &Client{
		dynamic:    dynamicClient,
		clientSet:  clientSet,
		restConfig: config,
		apiConfig:  *apiConfig,
		log:        log,
	}, nil
}

func (c *Client) NewChartClient() *ChartClient {
	return &ChartClient{
		kube:      c,
		tempCache: ".tmp",
	}
}

type Option interface{}
type DeleteOption struct{}
type ApplyOption struct{}
type resourceFunc func(ctx context.Context, resource *resourceObject, option Option) error
type resourceObject struct {
	resource           schema.GroupVersionResource
	unstructuredObject *unstructured.Unstructured
}

func (c *Client) Apply(object string, option ApplyOption) error {
	fi, err := os.Stat(object)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return c.yamlToAction(object, c.apply, option)
	}

	dirEntries, err := os.ReadDir(object)
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		err = c.yamlToAction(filepath.Join(object, dirEntry.Name()), c.apply, option)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Delete(object string, option DeleteOption) error {
	fi, err := os.Stat(object)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return c.yamlToAction(object, c.delete, option)
	}

	dirEntries, err := os.ReadDir(object)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		err = c.yamlToAction(filepath.Join(object, dirEntry.Name()), c.delete, option)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) yamlToAction(yamlFile string, action resourceFunc, option Option) error {
	fr, err := os.Open(yamlFile)
	if err != nil {
		return err
	}
	defer fr.Close()
	fi, err := fr.Stat()
	if err != nil {
		return err
	}

	ctx := context.TODO()
	decoder := yamlutil.NewYAMLOrJSONDecoder(fr, int(fi.Size()))

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if len(rawObj.Raw) == 0 {
			continue
		}

		obj, gvk, err := unstructured.UnstructuredJSONScheme.Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}

		err = c.refreshResource(gvk.Group)
		if err != nil {
			return err
		}

		mapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		mapper, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}
		resourceObj := &resourceObject{
			resource:           mapping.Resource,
			unstructuredObject: &unstructured.Unstructured{Object: mapper},
		}
		err = action(ctx, resourceObj, option)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) apply(ctx context.Context, object *resourceObject, option Option) error {
	if object.unstructuredObject.GetNamespace() == "" {
		return c.createClusterResource(ctx, object, option)
	}
	return c.createNamespacedResource(ctx, object, option)
}

func (c *Client) delete(ctx context.Context, object *resourceObject, option Option) error {
	if object.unstructuredObject.GetNamespace() == "" {
		return c.deleteClusterResource(ctx, &resourceObject{}, option.(DeleteOption))
	}
	return c.deleteNamespaceResource(ctx, object, option.(DeleteOption))
}

func (c *Client) createClusterResource(ctx context.Context, object *resourceObject, option Option) error {
	name := object.unstructuredObject.GetName()
	// kind := object.unstructuredObject.GetKind()
	_, err := c.dynamic.Resource(object.resource).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		_, err = c.dynamic.Resource(object.resource).Create(ctx, object.unstructuredObject, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	unstructuredYAML, err := sigyaml.Marshal(object.unstructuredObject)
	if err != nil {
		return err
	}

	force := true
	_, err = c.dynamic.Resource(object.resource).Patch(ctx, name, types.ApplyPatchType, unstructuredYAML, metav1.PatchOptions{
		FieldManager: name,
		Force:        &force,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) createNamespacedResource(ctx context.Context, object *resourceObject, option Option) error {
	name := object.unstructuredObject.GetName()
	namespace := object.unstructuredObject.GetNamespace()

	_, err := c.dynamic.Resource(object.resource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		_, err = c.clientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			return err
		}

		if errors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err = c.clientSet.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		}
		_, err = c.dynamic.Resource(object.resource).Namespace(namespace).Create(ctx, object.unstructuredObject, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	unstructuredYAML, err := sigyaml.Marshal(object.unstructuredObject)
	if err != nil {
		return err
	}

	force := true
	_, err = c.dynamic.Resource(object.resource).Patch(ctx, name, types.ApplyPatchType, unstructuredYAML, metav1.PatchOptions{
		FieldManager: name,
		Force:        &force,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) deleteClusterResource(ctx context.Context, object *resourceObject, option DeleteOption) error {
	name := object.unstructuredObject.GetName()
	_, err := c.dynamic.Resource(object.resource).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		return nil
	}
	return c.dynamic.Resource(object.resource).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) deleteNamespaceResource(ctx context.Context, object *resourceObject, option DeleteOption) error {
	name := object.unstructuredObject.GetName()
	namespace := object.unstructuredObject.GetNamespace()
	_, err := c.dynamic.Resource(object.resource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		return nil
	}
	return c.dynamic.Resource(object.resource).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) refreshResource(name string) error {
	restMapperRes, err := restmapper.GetAPIGroupResources(c.clientSet.Discovery())
	if err != nil {
		return err
	}

	c.restMapper = restmapper.NewDiscoveryRESTMapper(restMapperRes)
	return nil
}

func (c *Client) GetNodes() {
	nodes, err := c.clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("==>:", err)
		return
	}

	for _, n := range nodes.Items {
		fmt.Println(n.Name)
	}
}
