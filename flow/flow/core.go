package flow

import ctx "context"

type WrapF struct {
	f func(ctx ctx.Context) error

	ctx *Context
}

func WrapFunc(ctx ctx.Context, f func(ctx ctx.Context) error) *WrapF {
	return &WrapF{
		f:   f,
		ctx: newContext(ctx),
	}
}

func (wf *WrapF) UseBefore(f func(c *Context) error) {
	wf.ctx.addBeforeHandler(f)
}

func (wf *WrapF) UseBehind(f func(c *Context) error) {
	wf.ctx.addBehindHandler(f)
}

func (wf *WrapF) Handle() error {
	var err error
	wf.ctx.handlers = append(wf.ctx.handlers, wf.ctx.beforeHandlers...)
	wf.ctx.handlers = append(wf.ctx.handlers, func(c *Context) error {
		err := wf.f(wf.ctx)
		return err
	})
	wf.ctx.handlers = append(wf.ctx.handlers, wf.ctx.behindHandlers...)

	if len(wf.ctx.handlers) > 0 {
		err = wf.ctx.Next()
	}
	wf.ctx.Reset()

	return err
}
