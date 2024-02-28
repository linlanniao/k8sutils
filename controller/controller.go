package controller

import (
	"context"

	"k8s.io/klog/v2"
)

type Controller struct {
	resourceHandlers []ResourceHandler
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) AddResourceHandler(handler ResourceHandler) {
	c.resourceHandlers = append(c.resourceHandlers, handler)
}

func (c *Controller) Start(ctx context.Context) error {
	klog.Info("starting controller")

	// start the controller
	stop := make(chan struct{})
	defer close(stop)
	for _, h := range c.resourceHandlers {
		h := h
		go h.Run(stop)
	}

	// Wait forever
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
