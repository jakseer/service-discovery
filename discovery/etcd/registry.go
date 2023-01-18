package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"service-discovery/discovery"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	_ discovery.Registrar = (*Registry)(nil)
)

// Option is etcd registry option.
type Option func(o *options)

type options struct {
	namespace string
	ttl       time.Duration
}

// RegisterTTL register ttl
func RegisterTTL(ttl time.Duration) Option {
	return func(o *options) { o.ttl = ttl }
}

// Namespace registry namespace
func Namespace(ns string) Option {
	return func(o *options) { o.namespace = ns }
}

// Registry is etcd registry.
type Registry struct {
	opts   *options
	client *clientv3.Client
	lease  clientv3.Lease
}

// New creates etcd registry
func New(client *clientv3.Client, opts ...Option) (r *Registry) {
	op := &options{
		namespace: "/services",
		ttl:       time.Second * 15,
	}
	for _, o := range opts {
		o(op)
	}
	return &Registry{
		opts:   op,
		client: client,
	}
}

func (r *Registry) Register(ctx context.Context, service *discovery.ServiceInstance) error {
	vb, err := json.Marshal(service)
	if err != nil {
		return err
	}

	if r.lease != nil {
		r.lease.Close()
	}
	r.lease = clientv3.NewLease(r.client)

	key := fmt.Sprintf("%s/%s/%s", r.opts.namespace, service.Name, service.ID)
	leaseID, err := r.registerWithKV(ctx, key, string(vb))
	if err != nil {
		return err
	}

	go r.heartBeat(ctx, leaseID)
	return nil
}

func (r *Registry) Deregister(ctx context.Context, service *discovery.ServiceInstance) error {
	defer func() {
		if r.lease != nil {
			r.lease.Close()
		}
	}()
	key := fmt.Sprintf("%s/%s/%s", r.opts.namespace, service.Name, service.ID)
	_, err := r.client.Delete(ctx, key)
	return err
}

func (r *Registry) GetService(ctx context.Context, name string) ([]*discovery.ServiceInstance, error) {
	key := fmt.Sprintf("%s/%s", r.opts.namespace, name)
	resp, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	items := make([]*discovery.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var si discovery.ServiceInstance
		err := json.Unmarshal(kv.Value, &si)
		if err != nil {
			return nil, err
		}
		items = append(items, &si)
	}
	return items, nil
}

func (r *Registry) Watcher(ctx context.Context, serviceName string) (discovery.Watcher, error) {
	return newWatcher(ctx, r.opts.namespace, serviceName, r.client)
}

func (r *Registry) registerWithKV(ctx context.Context, key string, value string) (clientv3.LeaseID, error) {
	grant, err := r.lease.Grant(ctx, int64(r.opts.ttl.Seconds()))
	if err != nil {
		return 0, err
	}
	_, err = r.client.Put(ctx, key, value, clientv3.WithLease(grant.ID))
	if err != nil {
		return 0, err
	}
	return grant.ID, nil
}

func (r *Registry) heartBeat(ctx context.Context, leaseID clientv3.LeaseID) {
	ch, err := r.client.KeepAlive(ctx, leaseID)
	if err != nil {
		return
	}

	for {
		select {
		case keepaliveResp, ok := <-ch:
			if !ok {
				return
			}
			log.Default().Printf("%s keepalive resp=%+v", time.Now().String(), keepaliveResp)
		case <-ctx.Done():
			return
		}
	}
}
