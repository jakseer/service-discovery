package transport

import (
	"context"
	"net"
	"service-discovery/discovery"
	"time"
)

type Server struct {
	instance       discovery.Registrar
	currentService *discovery.ServiceInstance
}

const (
	tickTimeDuration   = time.Second * 60
	healthCheckTimeout = time.Second * 3
)

func NewServer(ctx context.Context, service discovery.ServiceInstance, instance discovery.Registrar) (*Server, error) {
	cli := &Server{
		instance:       instance,
		currentService: &service,
	}

	go func() {
		ticker := time.NewTicker(tickTimeDuration)
		for {
			select {
			case <-ticker.C:
				cli.healthCheck(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return cli, nil
}

// Register 注册服务
func (c *Server) Register(ctx context.Context) error {
	return c.instance.Register(ctx, c.currentService)
}

// Deregister 注销服务
func (c *Server) Deregister(ctx context.Context) error {
	return c.instance.Deregister(ctx, c.currentService)
}

// healthCheck 检查服务是否健康
func (c *Server) healthCheck(ctx context.Context) error {
	serviceList, err := c.instance.GetService(ctx, c.currentService.Name)
	if err != nil {
		return err
	}

	for _, v := range serviceList {
		if err = c.healthCheckOne(ctx, *v); err != nil {
			return err
		}
	}

	return nil
}

func (c *Server) healthCheckOne(ctx context.Context, service discovery.ServiceInstance) error {
	var validEndpoints []*discovery.EndPointInstance
	for _, v := range service.Endpoints {
		dialEndpointConn, err := net.DialTimeout("tcp", v.Endpoint, healthCheckTimeout)
		if err != nil {
			continue
		}
		dialEndpointConn.Close()
		v.HealthyAt = time.Now().Unix()
		validEndpoints = append(validEndpoints, v)
	}
	service.Endpoints = validEndpoints

	return c.instance.Register(ctx, &service)
}
