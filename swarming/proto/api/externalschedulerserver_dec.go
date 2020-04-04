// Code generated by svcdec; DO NOT EDIT.

package apipb

import (
	"context"

	proto "github.com/golang/protobuf/proto"
)

type DecoratedExternalScheduler struct {
	// Service is the service to decorate.
	Service ExternalSchedulerServer
	// Prelude is called for each method before forwarding the call to Service.
	// If Prelude returns an error, then the call is skipped and the error is
	// processed via the Postlude (if one is defined), or it is returned directly.
	Prelude func(ctx context.Context, methodName string, req proto.Message) (context.Context, error)
	// Postlude is called for each method after Service has processed the call, or
	// after the Prelude has returned an error. This takes the the Service's
	// response proto (which may be nil) and/or any error. The decorated
	// service will return the response (possibly mutated) and error that Postlude
	// returns.
	Postlude func(ctx context.Context, methodName string, rsp proto.Message, err error) error
}

func (s *DecoratedExternalScheduler) AssignTasks(ctx context.Context, req *AssignTasksRequest) (rsp *AssignTasksResponse, err error) {
	if s.Prelude != nil {
		var newCtx context.Context
		newCtx, err = s.Prelude(ctx, "AssignTasks", req)
		if err == nil {
			ctx = newCtx
		}
	}
	if err == nil {
		rsp, err = s.Service.AssignTasks(ctx, req)
	}
	if s.Postlude != nil {
		err = s.Postlude(ctx, "AssignTasks", rsp, err)
	}
	return
}

func (s *DecoratedExternalScheduler) GetCancellations(ctx context.Context, req *GetCancellationsRequest) (rsp *GetCancellationsResponse, err error) {
	if s.Prelude != nil {
		var newCtx context.Context
		newCtx, err = s.Prelude(ctx, "GetCancellations", req)
		if err == nil {
			ctx = newCtx
		}
	}
	if err == nil {
		rsp, err = s.Service.GetCancellations(ctx, req)
	}
	if s.Postlude != nil {
		err = s.Postlude(ctx, "GetCancellations", rsp, err)
	}
	return
}

func (s *DecoratedExternalScheduler) NotifyTasks(ctx context.Context, req *NotifyTasksRequest) (rsp *NotifyTasksResponse, err error) {
	if s.Prelude != nil {
		var newCtx context.Context
		newCtx, err = s.Prelude(ctx, "NotifyTasks", req)
		if err == nil {
			ctx = newCtx
		}
	}
	if err == nil {
		rsp, err = s.Service.NotifyTasks(ctx, req)
	}
	if s.Postlude != nil {
		err = s.Postlude(ctx, "NotifyTasks", rsp, err)
	}
	return
}

func (s *DecoratedExternalScheduler) GetCallbacks(ctx context.Context, req *GetCallbacksRequest) (rsp *GetCallbacksResponse, err error) {
	if s.Prelude != nil {
		var newCtx context.Context
		newCtx, err = s.Prelude(ctx, "GetCallbacks", req)
		if err == nil {
			ctx = newCtx
		}
	}
	if err == nil {
		rsp, err = s.Service.GetCallbacks(ctx, req)
	}
	if s.Postlude != nil {
		err = s.Postlude(ctx, "GetCallbacks", rsp, err)
	}
	return
}