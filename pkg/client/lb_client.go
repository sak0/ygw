package client

import (
	lbv1 "github.com/sak0/ygw/pkg/apis/loadbalance/v1"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func CALBPoolClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace string) *calbpclient {
	return &calbpclient{cl: cl, ns: namespace, plural: lbv1.CALBPPlural,
		codec: runtime.NewParameterCodec(scheme)}
}

type calbpclient struct {
	cl		*rest.RESTClient
	ns		string
	plural	string
	codec	runtime.ParameterCodec
}

func (f *calbpclient) Create(obj *lbv1.CAppLoadBalancePool) (*lbv1.CAppLoadBalancePool, error) {
	var result lbv1.CAppLoadBalancePool
	err := f.cl.Post().
		Namespace(f.ns).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *calbpclient) Update(obj *lbv1.CAppLoadBalancePool, name string) (*lbv1.CAppLoadBalancePool, error) {
	var result lbv1.CAppLoadBalancePool
	err := f.cl.Put().
		Namespace(f.ns).Resource(f.plural).
		Name(name).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *calbpclient) Delete(name string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}

func (f *calbpclient) Get(name string) (*lbv1.CAppLoadBalancePool, error) {
	var result lbv1.CAppLoadBalancePool
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *calbpclient) List(opts meta_v1.ListOptions) (*lbv1.CAppLoadBalancePoolList, error) {
	var result lbv1.CAppLoadBalancePoolList
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

// Create a new List watch for our TPR
func (f *calbpclient) NewListWatch() *cache.ListWatch {
	//return cache.NewListWatchFromClient(f.cl, f.plural, f.ns, fields.Everything())
	return cache.NewListWatchFromClient(f.cl, f.plural, meta_v1.NamespaceAll, fields.Everything())
}



func CALBClient(cl *rest.RESTClient, scheme *runtime.Scheme, namespace string) *calbclient {
	return &calbclient{cl: cl, ns: namespace, plural: lbv1.CALBPlural,
		codec: runtime.NewParameterCodec(scheme)}
}

type calbclient struct {
	cl		*rest.RESTClient
	ns		string
	plural	string
	codec	runtime.ParameterCodec
}

func (f *calbclient) Create(obj *lbv1.CAppLoadBalance) (*lbv1.CAppLoadBalance, error) {
	var result lbv1.CAppLoadBalance
	err := f.cl.Post().
		Namespace(f.ns).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *calbclient) Update(obj *lbv1.CAppLoadBalance, name string) (*lbv1.CAppLoadBalance, error) {
	var result lbv1.CAppLoadBalance
	err := f.cl.Put().
		Namespace(f.ns).Resource(f.plural).
		Name(name).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *calbclient) Delete(name string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}

func (f *calbclient) Get(name string) (*lbv1.CAppLoadBalance, error) {
	var result lbv1.CAppLoadBalance
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *calbclient) List(opts meta_v1.ListOptions) (*lbv1.CAppLoadBalanceList, error) {
	var result lbv1.CAppLoadBalanceList
	err := f.cl.Get().
		Namespace(f.ns).Resource(f.plural).
		VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

// Create a new List watch for our TPR
func (f *calbclient) NewListWatch() *cache.ListWatch {
	//return cache.NewListWatchFromClient(f.cl, f.plural, f.ns, fields.Everything())
	return cache.NewListWatchFromClient(f.cl, f.plural, meta_v1.NamespaceAll, fields.Everything())
}



func NewLBClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := lbv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	
	config := *cfg
	config.GroupVersion = &lbv1.SchemeGroupVersion
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