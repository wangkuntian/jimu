package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// PingServer 参考服务：echo 应答，演示免 protoc 的 ServiceDesc 注册方式。
// 业务模块仿照本文件注册自己的服务；消息类型先用 wrapperspb/emptypb 组合，
// 需要自定义结构时再引入 protoc-gen-go 生成代码。
type PingServer interface {
	Ping(ctx context.Context, in *wrapperspb.StringValue) (*wrapperspb.StringValue, error)
}

// RegisterPingServer 注册 Ping 服务到 gRPC server。
func RegisterPingServer(s *grpc.Server, srv PingServer) {
	s.RegisterService(&pingServiceDesc, srv)
}

type pingService struct{}

func (pingService) Ping(ctx context.Context, in *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	if in == nil || in.Value == "" {
		return nil, status.Error(codes.InvalidArgument, "msg required")
	}
	return wrapperspb.String("pong:" + in.Value), nil
}

var pingServiceDesc = grpc.ServiceDesc{
	ServiceName: "jimu.v1.Ping",
	HandlerType: (*PingServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(wrapperspb.StringValue)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					ps, ok := srv.(PingServer)
					if !ok {
						return nil, status.Error(codes.Internal, "server does not implement PingServer")
					}
					return ps.Ping(ctx, in)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/jimu.v1.Ping/Ping"}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					msg, ok := req.(*wrapperspb.StringValue)
					if !ok {
						return nil, status.Error(codes.Internal, "invalid request type")
					}
					ps, ok := srv.(PingServer)
					if !ok {
						return nil, status.Error(codes.Internal, "server does not implement PingServer")
					}
					return ps.Ping(ctx, msg)
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "jimu/v1/ping.proto",
}
