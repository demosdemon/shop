package pool

import (
	"context"

	"github.com/demosdemon/multierrgroup"
)

type Context struct {
	context context.Context
	group   multierrgroup.Group
}

func New(ctx context.Context) *Context {
	return &Context{context: ctx}
}

func (ctx *Context) Go(fn func(ctx context.Context) error) {
	ctx.group.Go(func() error {
		return fn(ctx.context)
	})
}

func (ctx *Context) Wait() error {
	return ctx.group.Wait()
}
