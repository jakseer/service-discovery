package transport

import (
	"context"
	"math/rand"
	"service-discovery/discovery"
)

type Client struct {
	instance       discovery.Registrar
	currentService *discovery.ServiceInstance
}

func NewClient(ctx context.Context, service discovery.ServiceInstance, instance discovery.Registrar) (*Client, error) {
	cli := &Client{
		currentService: &service,
		instance:       instance,
	}

	return cli, nil
}

// GetEndpoints 获取所有的endpoint
func (c *Client) GetEndpoints(ctx context.Context, name string) ([]*discovery.EndPointInstance, error) {
	serviceList, err := c.instance.GetService(ctx, name)
	if err != nil {
		return nil, err
	}

	var ret []*discovery.EndPointInstance
	for _, service := range serviceList {
		for _, endPoint := range service.Endpoints {
			ret = append(ret, endPoint)
		}
	}
	return ret, nil
}

// PickOneEndpoint 根据 weight 挑选一个endpoint
func (c *Client) PickOneEndpoint(ctx context.Context, name string) (*discovery.EndPointInstance, error) {
	endpointList, err := c.GetEndpoints(ctx, name)
	if err != nil {
		return nil, err
	}

	var totalWeight int32
	for _, v := range endpointList {
		totalWeight += v.Weight
	}

	randNum := rand.Int31n(int32(totalWeight))
	for _, v := range endpointList {
		randNum -= v.Weight
		if randNum <= 0 {
			return v, nil
		}
	}

	return nil, nil
}
