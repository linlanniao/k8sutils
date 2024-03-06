package controller

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"
)

type Handler interface {
	Run(stopCh chan struct{})
}

type Controller struct {
	Handlers []Handler
}

type Option func(*Controller)

func NewController(opts ...Option) *Controller {
	c := &Controller{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithHandlers(handlers ...Handler) Option {
	return func(c *Controller) {
		length := len(handlers)
		if length == 0 {
			return
		}
		if c.Handlers == nil {
			c.Handlers = make([]Handler, length)
		}
		for i := 0; i < length; i++ {
			c.Handlers[i] = handlers[i]
		}
	}
}

func (c *Controller) AddHandler(handler Handler) {
	c.Handlers = append(c.Handlers, handler)
}

func (c *Controller) Start(ctx context.Context) error {
	if len(c.Handlers) == 0 {
		return fmt.Errorf("no handler")
	}

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
