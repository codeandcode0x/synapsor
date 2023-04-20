package grpc

import (
	"fmt"
	"io"
	"os"
	"strconv"
	logging "synapsor/pkg/core/log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	grpcRetryTimes              string
	grpcRetrySleepTimes         string
	grpcRetryTimesInt           int = 10
	grpcRetrySleepTimesInt      int = 100
	clientStreamDescForProxying     = &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}
)

type protoCodec struct{}

// codec
func Codec() grpc.Codec {
	return CodecWithParent(&protoCodec{})
}

// codec with parent
func CodecWithParent(fallback grpc.Codec) grpc.Codec {
	return &rawCodec{fallback}
}

// raw codec
type rawCodec struct {
	parentCodec grpc.Codec
}

// frame struct
type frame struct {
	payload []byte
}

// marshal
func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*frame)
	if !ok {
		return c.parentCodec.Marshal(v)
	}
	return out.payload, nil

}

// unmarshal
func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*frame)
	if !ok {
		return c.parentCodec.Unmarshal(data, v)
	}
	dst.payload = data
	return nil
}

// string func
func (c *rawCodec) String() string {
	return fmt.Sprintf("proxy>%s", c.parentCodec.String())
}

// marshal
func (protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

// unmarshal
func (protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

// string func()
func (protoCodec) String() string {
	return "proto"
}

// register service
func RegisterService(server *grpc.Server, director StreamDirector, serviceName string, methodNames ...string) {
	streamer := &handler{director}
	fakeDesc := &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*interface{})(nil),
	}
	for _, m := range methodNames {
		streamDesc := grpc.StreamDesc{
			StreamName:    m,
			Handler:       streamer.handler,
			ServerStreams: true,
			ClientStreams: true,
		}
		fakeDesc.Streams = append(fakeDesc.Streams, streamDesc)
	}
	server.RegisterService(fakeDesc, streamer)
}

// transparent handler
func TransparentHandler(director StreamDirector) grpc.StreamHandler {
	streamer := &handler{director}
	return streamer.handler
}

// handler struct
type handler struct {
	director StreamDirector
}

// handler func
func (s *handler) handler(srv interface{}, serverStream grpc.ServerStream) error {
	// get grpc retry times
	grpcRetryTimes = os.Getenv("GRPC_RETRY_TIMES")
	if grpcRetryTimes != "" {
		grpcRetryTimesInt, _ = strconv.Atoi(grpcRetryTimes)
	}
	// get grpc retry sleep times
	grpcRetrySleepTimes = os.Getenv("GRPC_RETRY_SLEEP_TIMES")
	if grpcRetrySleepTimes != "" {
		grpcRetrySleepTimesInt, _ = strconv.Atoi(grpcRetrySleepTimes)
	}

	now := time.Now()
	timeStart := now.UnixMilli()

	fullMethodName, ok := grpc.MethodFromServerStream(serverStream)
	if !ok {
		return status.Errorf(codes.Internal, "lowLevelServerStream not exists in context")
	}

	outgoingCtx, backendConn, conn, err := s.director(serverStream.Context(), fullMethodName)
	if err != nil {
		c := 0
		for ; c < grpcRetryTimesInt; c++ {
			outgoingCtx, backendConn, conn, err = s.director(serverStream.Context(), fullMethodName)
			if err == nil {
				break
			}
		}
		if c >= grpcRetryTimesInt {
			logging.ERROR.Error("-----------------------  create stream director:", err.Error())
			return err
		}
	}

	defer conn.Close()

	clientCtx, clientCancel := context.WithCancel(outgoingCtx)
	// TODO(mwitkow): Add a `forwarded` header to metadata, https://en.wikipedia.org/wiki/X-Forwarded-For.
	clientStream, err := grpc.NewClientStream(clientCtx, clientStreamDescForProxying, backendConn, fullMethodName)
	if err != nil {
		c := 0
		for ; c < grpcRetryTimesInt; c++ {
			logging.ERROR.Error("-----------------------  create stream error:", err.Error())
			clientStream, err = grpc.NewClientStream(clientCtx, clientStreamDescForProxying, backendConn, fullMethodName)
			if err != nil {
				SleepTime(grpcRetrySleepTimesInt)
			} else {
				break
			}
		}
		// return err
		if c >= grpcRetryTimesInt {
			return err
		}
	}

	s2cErrChan := s.forwardServerToClient(serverStream, clientStream)
	c2sErrChan := s.forwardClientToServer(clientStream, serverStream)
	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				clientStream.CloseSend()
				// break
			} else {
				clientCancel()
				logging.ERROR.Error("-----------------------  create stream S2C:", codes.Internal, " failed proxying s2c: ", s2cErr)
				return status.Errorf(codes.Internal, "failed proxying s2c: %v", s2cErr)
			}
		case c2sErr := <-c2sErrChan:
			endNow := time.Now()
			timeEnd := endNow.UnixMilli()
			logging.Log.Info("c 2 s", timeStart, timeEnd, "request time spend: ", timeEnd-timeStart, " average request time:")
			serverStream.SetTrailer(clientStream.Trailer())
			if c2sErr != io.EOF {
				logging.ERROR.Error("-----------------------  create stream S2C:", codes.Internal, " failed proxying s2c: ", c2sErr)
				return c2sErr
			}
			return nil
		}
	}
	return status.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
}

// forward client to server
func (s *handler) forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &frame{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				if err == io.EOF {
					ret <- err
					break
				}
				c := 0
				for ; c < grpcRetryTimesInt; c++ {
					err = src.RecvMsg(f)
					if err != nil {
						SleepTime(grpcRetrySleepTimesInt)
					} else {
						break
					}
				}
				if c >= grpcRetryTimesInt {
					ret <- err
					break
				}
			}
			if i == 0 {
				md, err := src.Header()
				if err != nil {
					ret <- err
					break
				}
				if err := dst.SendHeader(md); err != nil {
					ret <- err
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				c := 0
				for ; c < grpcRetryTimesInt; c++ {
					err = dst.SendMsg(f)
					if err != nil {
						SleepTime(grpcRetrySleepTimesInt)
					} else {
						break
					}
				}
				if c >= grpcRetryTimesInt {
					ret <- err
					break
				}
			}
		}
	}()
	return ret
}

// forward server to client
func (s *handler) forwardServerToClient(src grpc.ServerStream, dst grpc.ClientStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &frame{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				if err == io.EOF {
					ret <- err
					break
				}
				c := 0
				for ; c < grpcRetryTimesInt; c++ {
					err = src.RecvMsg(f)
					if err != nil {
						SleepTime(grpcRetrySleepTimesInt)
					} else {
						break
					}
				}
				if c >= grpcRetryTimesInt {
					ret <- err
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				c := 0
				for ; c < grpcRetryTimesInt; c++ {
					err = dst.SendMsg(f)
					if err != nil {
						SleepTime(grpcRetrySleepTimesInt)
					} else {
						break
					}
				}
				if c >= grpcRetryTimesInt {
					ret <- err
					break
				}
			}
		}
	}()
	return ret
}

// sleep times
func SleepTime(sleepTimes int) {
	for i := 0; i < sleepTimes/100; i++ {
		time.Sleep(time.Millisecond * 100)
	}
}
