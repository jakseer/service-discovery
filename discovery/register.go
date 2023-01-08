package discovery

import "context"

type ServiceInstance struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Endpoints []*EndPointInstance `json:"endpoints"`
}

type EndPointInstance struct {
	Endpoint  string `json:"endpoint"`
	Weight    int32  `json:"weight"`
	HealthyAt int64  `json:"healthy_at"`
}

type Watcher interface {
	Watch(ctx context.Context) <-chan struct{}

	Stop() error
}

type Registrar interface {
	Register(ctx context.Context, service *ServiceInstance) error

	Deregister(ctx context.Context, service *ServiceInstance) error

	GetService(ctx context.Context, serviceName string) ([]*ServiceInstance, error)

	Watcher(ctx context.Context, serviceName string) (Watcher, error)
}
