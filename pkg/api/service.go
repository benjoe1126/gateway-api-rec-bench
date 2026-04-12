package api

import (
	"context"
	"encoding/json"
	"onlab-bm/pkg/patch"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

var ServiceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}

type ServiceApi struct {
	client *dynamic.DynamicClient
}

func NewServiceApi(client *dynamic.DynamicClient) *ServiceApi {
	return &ServiceApi{client: client}
}

func (s *ServiceApi) Get(ctx context.Context, name, namespace string) (GWV1Resource, error) {
	unstruct, err := s.client.Resource(ServiceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	service := &v1.Service{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), service)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (s *ServiceApi) List(ctx context.Context, namespace string) ([]GWV1Resource, error) {
	list, err := s.client.Resource(ServiceGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	services := make([]GWV1Resource, 0, len(list.Items))
	for _, item := range list.Items {
		service := &v1.Service{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, service)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}
func (s *ServiceApi) Create(ctx context.Context, service GWV1Resource) error {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(service)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: unstruct}
	if _, err = s.client.Resource(ServiceGVR).Namespace(service.GetNamespace()).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (s *ServiceApi) Patch(ctx context.Context, name, namespace string, patches []patch.JsonPatch) error {
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return err
	}
	_, err = s.client.Resource(ServiceGVR).Namespace(namespace).Patch(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (s *ServiceApi) Delete(ctx context.Context, name, namespace string) error {
	return s.client.Resource(ServiceGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
