package controller

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"
)

type Controller interface {
	Run(stopCh chan struct{})
	Namespace() string
}

type MasterController struct {
	controller []Controller
}

type Option func(*MasterController)

func NewMasterController(opts ...Option) *MasterController {
	c := &MasterController{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithHandlers(controllers ...Controller) Option {
	return func(c *MasterController) {
		length := len(controllers)
		if length == 0 {
			return
		}
		if c.controller == nil {
			c.controller = make([]Controller, length)
		}
		for i := 0; i < length; i++ {
			c.controller[i] = controllers[i]
		}
	}
}

func (c *MasterController) AddController(handler Controller) {
	c.controller = append(c.controller, handler)
}

func (c *MasterController) Run(ctx context.Context) error {
	if len(c.controller) == 0 {
		return fmt.Errorf("no handler")
	}

	klog.Info("starting controller")

	// start the controller
	stop := make(chan struct{})
	defer close(stop)
	for _, h := range c.controller {
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

// TODO MasterController how to get the namespace?
