// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.17.3
// source: proto/v1/service.proto

package v1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AmeshServiceClient is the client API for AmeshService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AmeshServiceClient interface {
	StreamPlugins(ctx context.Context, in *PluginsRequest, opts ...grpc.CallOption) (AmeshService_StreamPluginsClient, error)
}

type ameshServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAmeshServiceClient(cc grpc.ClientConnInterface) AmeshServiceClient {
	return &ameshServiceClient{cc}
}

func (c *ameshServiceClient) StreamPlugins(ctx context.Context, in *PluginsRequest, opts ...grpc.CallOption) (AmeshService_StreamPluginsClient, error) {
	stream, err := c.cc.NewStream(ctx, &AmeshService_ServiceDesc.Streams[0], "/ai.api7.amesh.proto.v1.AmeshService/StreamPlugins", opts...)
	if err != nil {
		return nil, err
	}
	x := &ameshServiceStreamPluginsClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AmeshService_StreamPluginsClient interface {
	Recv() (*PluginsResponse, error)
	grpc.ClientStream
}

type ameshServiceStreamPluginsClient struct {
	grpc.ClientStream
}

func (x *ameshServiceStreamPluginsClient) Recv() (*PluginsResponse, error) {
	m := new(PluginsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// AmeshServiceServer is the server API for AmeshService service.
// All implementations must embed UnimplementedAmeshServiceServer
// for forward compatibility
type AmeshServiceServer interface {
	StreamPlugins(*PluginsRequest, AmeshService_StreamPluginsServer) error
	mustEmbedUnimplementedAmeshServiceServer()
}

// UnimplementedAmeshServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAmeshServiceServer struct {
}

func (UnimplementedAmeshServiceServer) StreamPlugins(*PluginsRequest, AmeshService_StreamPluginsServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamPlugins not implemented")
}
func (UnimplementedAmeshServiceServer) mustEmbedUnimplementedAmeshServiceServer() {}

// UnsafeAmeshServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AmeshServiceServer will
// result in compilation errors.
type UnsafeAmeshServiceServer interface {
	mustEmbedUnimplementedAmeshServiceServer()
}

func RegisterAmeshServiceServer(s grpc.ServiceRegistrar, srv AmeshServiceServer) {
	s.RegisterService(&AmeshService_ServiceDesc, srv)
}

func _AmeshService_StreamPlugins_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(PluginsRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AmeshServiceServer).StreamPlugins(m, &ameshServiceStreamPluginsServer{stream})
}

type AmeshService_StreamPluginsServer interface {
	Send(*PluginsResponse) error
	grpc.ServerStream
}

type ameshServiceStreamPluginsServer struct {
	grpc.ServerStream
}

func (x *ameshServiceStreamPluginsServer) Send(m *PluginsResponse) error {
	return x.ServerStream.SendMsg(m)
}

// AmeshService_ServiceDesc is the grpc.ServiceDesc for AmeshService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AmeshService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ai.api7.amesh.proto.v1.AmeshService",
	HandlerType: (*AmeshServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamPlugins",
			Handler:       _AmeshService_StreamPlugins_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/v1/service.proto",
}
