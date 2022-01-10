package rpc

import (
	context "context"
	"errors"
	"fmt"
	reflect "reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (e *NotStartedError) Error() string {
	return fmt.Sprintf("some processes were not started (%s)", strings.Join(e.ProcessNames, ", "))
}

// ClientErrors wraps the gRPC client so that it returns errors that
// can be used with errors.As to extract the error details
// as added by ServerErrors. Use this to wrap the result of grpc.Dial
// before passing to NewGopmClient.
//
// The returned errors can also be used with errors.Is to
// check for status codes by using a CodeError instance.
func ClientErrors(cc grpc.ClientConnInterface) grpc.ClientConnInterface {
	return &clientErrorConn{cc}
}

type clientErrorConn struct {
	grpc.ClientConnInterface
}

func (cc *clientErrorConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	err := cc.ClientConnInterface.Invoke(ctx, method, args, reply, opts...)
	if err == nil {
		return nil
	}
	return &serverError{
		status: status.Convert(err),
	}
}

type serverError struct {
	status *status.Status
}

func (err *serverError) Error() string {
	return fmt.Sprintf("rpc error: %v", err.status.Message())
}

func (err *serverError) Is(e error) bool {
	e1, ok := e.(CodeError)
	if !ok {
		return false
	}
	return err.status.Code() == e1.Code
}

func (err *serverError) As(e interface{}) bool {
	if e, ok := e.(*CodeError); ok {
		e.Code = err.status.Code()
		return true
	}
	ev := reflect.ValueOf(e).Elem()
	for _, d := range err.status.Details() {
		dv := reflect.ValueOf(d)
		if dv.Type().AssignableTo(ev.Type()) {
			ev.Set(dv)
			return true
		}
	}
	return false
}

// ErrNotFound represents a gRPC not-found error.
var ErrNotFound = CodeError{Code: codes.NotFound}

// CodeError represents a gRPC status code error.
// This is defined mostly so that it's possible to use
// errors.Is, for example:
//
//	if errors.Is(err, rpc.ErrNotFound)
type CodeError struct {
	Code codes.Code
}

func (e CodeError) Error() string {
	return e.Code.String()
}

func (e CodeError) StatusCode() codes.Code {
	return e.Code
}

// ServerErrors returns a server option that converts protobuf-marshalable errors
// into errors that can be passed back over the wire and reassembled with ClientErrors.
// Pass this to grpc.NewServer before registering the gRPC server implementation.
//
// Any error returned from an RPC endpoint that implements proto.Message
// will be added as error details to the gRPC error.
func ServerErrors() grpc.ServerOption {
	return grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		resp, err = handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		var m proto.Message
		if !errors.As(err, &m) {
			return resp, err
		}
		// The error can be represented as a protobuf message, so add it to the error details.
		code := codes.Unknown
		if m, ok := m.(interface{ StatusCode() codes.Code }); ok {
			code = m.StatusCode()
		}
		st, err := status.New(code, err.Error()).WithDetails(m)
		if err != nil {
			panic(fmt.Errorf("cannot encode message details: %v", err))
		}
		return resp, st.Err()
	})
}
