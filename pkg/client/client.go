package client

import (
	crdv1 "github.com/sak0/ygw/pkg/apis/external/v1"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func PoolClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace string) *poolclient {
	return &poolclient{cl: cl, ns: namespace, plural: crdv1.EXPPlural,
		codec: runtime.NewParameterCodec(scheme)}
}

type poolclient struct {
	cl		*rest.RESTClient
	ns		string
	plural	string
	codec	runtime.ParameterCodec
}

func (f *poolclient) Create(obj *crdv1.ExternalNatPool) (*crdv1.ExternalNatPool, error) {
	var result crdv1.ExternalNatPool
	err := f.cl.Post().
		Namespace(f.ns).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *poolclient) Update(obj *crdv1.ExternalNatPool, name string) (*crdv1.ExternalNatPool, error) {
	var result crdv1.ExternalNatPool
	err := f.cl.Put().
		Namespace(f.ns).Resource(f.plural).
		Name(name).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *poolclient) Delete(name string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}

func (f *poolclient) Get(name string) (*crdv1.ExternalNatPool, error) {
	var result crdv1.ExternalNatPool
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *poolclient) List(opts meta_v1.ListOptions) (*crdv1.ExternalNatPoolList, error) {
	var result crdv1.ExternalNatPoolList
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

// Create a new List watch for our TPR
func (f *poolclient) NewListWatch() *cache.ListWatch {
	//return cache.NewListWatchFromClient(f.cl, f.plural, f.ns, fields.Everything())
	return cache.NewListWatchFromClient(f.cl, f.plural, meta_v1.NamespaceAll, fields.Everything())
}

func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := crdv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	
	config := *cfg
	config.GroupVersion = &crdv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}
	return client, scheme, nil
}
