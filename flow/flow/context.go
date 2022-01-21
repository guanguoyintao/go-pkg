package flow

import (
	ctx "context"
	"math"
)

const abort = math.MaxInt32 - 10000
const PAYLOADKEY = "payload"

type Context struct {
	ctx.Context
	offset         int
	beforeHandlers []func(ctx *Context) error
	behindHandlers []func(ctx *Context) error
	handlers       []func(ctx *Context) error
}

func newContext(ctx ctx.Context) *Context {
	return &Context{
		Context:        ctx,
		offset:         -1,
		beforeHandlers: make([]func(*Context) error, 0, 10),
		behindHandlers: make([]func(*Context) error, 0, 10),
		handlers:       make([]func(*Context) error, 0, 20),
	}
}

func (ctx *Context) Next() error {
	ctx.offset++
	s := len(ctx.handlers)
	for ; ctx.offset < s; ctx.offset++ {
		if !ctx.isAbort() {
			err := ctx.handlers[ctx.offset](ctx)
			if err != nil {
				return err
			}
		} else {
			return nil
		}
	}

	return nil
}

func (ctx *Context) Reset() {
	ctx.offset = -1
	ctx.behindHandlers = ctx.handlers[:0]
	ctx.beforeHandlers = ctx.handlers[:0]
	ctx.handlers = ctx.handlers[:0]
}

func (ctx *Context) Abort() {
	ctx.offset = math.MaxInt32 - 10000
}

func (ctx *Context) isAbort() bool {
	if ctx.offset >= abort {
		return true
	}
	return false
}

func (ctx *Context) addBeforeHandler(f func(ctx *Context) error) {
	ctx.beforeHandlers = append(ctx.beforeHandlers, f)
}

func (ctx *Context) addBehindHandler(f func(ctx *Context) error) {
	ctx.behindHandlers = append(ctx.behindHandlers, f)
}
