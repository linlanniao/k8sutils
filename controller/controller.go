package controller

import (
	"context"

	"k8s.io/klog/v2"
)

type Controller struct {
	Handlers []*Handler
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) AddHandler(handler *Handler) {
	c.Handlers = append(c.Handlers, handler)
}

func (c *Controller) Start(ctx context.Context) error {
	klog.Info("starting controller")

	// start the controller
	stop := make(chan struct{})
	defer close(stop)
	for _, h := range c.Handlers {
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
