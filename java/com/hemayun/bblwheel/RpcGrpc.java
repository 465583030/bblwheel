package com.hemayun.bblwheel;

import static io.grpc.stub.ClientCalls.asyncUnaryCall;
import static io.grpc.stub.ClientCalls.asyncServerStreamingCall;
import static io.grpc.stub.ClientCalls.asyncClientStreamingCall;
import static io.grpc.stub.ClientCalls.asyncBidiStreamingCall;
import static io.grpc.stub.ClientCalls.blockingUnaryCall;
import static io.grpc.stub.ClientCalls.blockingServerStreamingCall;
import static io.grpc.stub.ClientCalls.futureUnaryCall;
import static io.grpc.MethodDescriptor.generateFullMethodName;
import static io.grpc.stub.ServerCalls.asyncUnaryCall;
import static io.grpc.stub.ServerCalls.asyncServerStreamingCall;
import static io.grpc.stub.ServerCalls.asyncClientStreamingCall;
import static io.grpc.stub.ServerCalls.asyncBidiStreamingCall;
import static io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall;
import static io.grpc.stub.ServerCalls.asyncUnimplementedStreamingCall;

/**
 */
@javax.annotation.Generated(
    value = "by gRPC proto compiler (version 1.0.2)",
    comments = "Source: rpc.proto")
public class RpcGrpc {

  private RpcGrpc() {}

  public static final String SERVICE_NAME = "bblwheel.Rpc";

  // Static method descriptors that strictly reflect the proto.
  @io.grpc.ExperimentalApi("https://github.com/grpc/grpc-java/issues/1901")
  public static final io.grpc.MethodDescriptor<com.hemayun.bblwheel.RpcOuterClass.Request,
      com.hemayun.bblwheel.RpcOuterClass.Response> METHOD_CALL =
      io.grpc.MethodDescriptor.create(
          io.grpc.MethodDescriptor.MethodType.UNARY,
          generateFullMethodName(
              "bblwheel.Rpc", "Call"),
          io.grpc.protobuf.ProtoUtils.marshaller(com.hemayun.bblwheel.RpcOuterClass.Request.getDefaultInstance()),
          io.grpc.protobuf.ProtoUtils.marshaller(com.hemayun.bblwheel.RpcOuterClass.Response.getDefaultInstance()));
  @io.grpc.ExperimentalApi("https://github.com/grpc/grpc-java/issues/1901")
  public static final io.grpc.MethodDescriptor<com.hemayun.bblwheel.RpcOuterClass.Message,
      com.hemayun.bblwheel.RpcOuterClass.Message> METHOD_CHANNEL =
      io.grpc.MethodDescriptor.create(
          io.grpc.MethodDescriptor.MethodType.BIDI_STREAMING,
          generateFullMethodName(
              "bblwheel.Rpc", "Channel"),
          io.grpc.protobuf.ProtoUtils.marshaller(com.hemayun.bblwheel.RpcOuterClass.Message.getDefaultInstance()),
          io.grpc.protobuf.ProtoUtils.marshaller(com.hemayun.bblwheel.RpcOuterClass.Message.getDefaultInstance()));

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static RpcStub newStub(io.grpc.Channel channel) {
    return new RpcStub(channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static RpcBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    return new RpcBlockingStub(channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary and streaming output calls on the service
   */
  public static RpcFutureStub newFutureStub(
      io.grpc.Channel channel) {
    return new RpcFutureStub(channel);
  }

  /**
   */
  public static abstract class RpcImplBase implements io.grpc.BindableService {

    /**
     */
    public void call(com.hemayun.bblwheel.RpcOuterClass.Request request,
        io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Response> responseObserver) {
      asyncUnimplementedUnaryCall(METHOD_CALL, responseObserver);
    }

    /**
     */
    public io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Message> channel(
        io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Message> responseObserver) {
      return asyncUnimplementedStreamingCall(METHOD_CHANNEL, responseObserver);
    }

    @java.lang.Override public io.grpc.ServerServiceDefinition bindService() {
      return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
          .addMethod(
            METHOD_CALL,
            asyncUnaryCall(
              new MethodHandlers<
                com.hemayun.bblwheel.RpcOuterClass.Request,
                com.hemayun.bblwheel.RpcOuterClass.Response>(
                  this, METHODID_CALL)))
          .addMethod(
            METHOD_CHANNEL,
            asyncBidiStreamingCall(
              new MethodHandlers<
                com.hemayun.bblwheel.RpcOuterClass.Message,
                com.hemayun.bblwheel.RpcOuterClass.Message>(
                  this, METHODID_CHANNEL)))
          .build();
    }
  }

  /**
   */
  public static final class RpcStub extends io.grpc.stub.AbstractStub<RpcStub> {
    private RpcStub(io.grpc.Channel channel) {
      super(channel);
    }

    private RpcStub(io.grpc.Channel channel,
        io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected RpcStub build(io.grpc.Channel channel,
        io.grpc.CallOptions callOptions) {
      return new RpcStub(channel, callOptions);
    }

    /**
     */
    public void call(com.hemayun.bblwheel.RpcOuterClass.Request request,
        io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Response> responseObserver) {
      asyncUnaryCall(
          getChannel().newCall(METHOD_CALL, getCallOptions()), request, responseObserver);
    }

    /**
     */
    public io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Message> channel(
        io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Message> responseObserver) {
      return asyncBidiStreamingCall(
          getChannel().newCall(METHOD_CHANNEL, getCallOptions()), responseObserver);
    }
  }

  /**
   */
  public static final class RpcBlockingStub extends io.grpc.stub.AbstractStub<RpcBlockingStub> {
    private RpcBlockingStub(io.grpc.Channel channel) {
      super(channel);
    }

    private RpcBlockingStub(io.grpc.Channel channel,
        io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected RpcBlockingStub build(io.grpc.Channel channel,
        io.grpc.CallOptions callOptions) {
      return new RpcBlockingStub(channel, callOptions);
    }

    /**
     */
    public com.hemayun.bblwheel.RpcOuterClass.Response call(com.hemayun.bblwheel.RpcOuterClass.Request request) {
      return blockingUnaryCall(
          getChannel(), METHOD_CALL, getCallOptions(), request);
    }
  }

  /**
   */
  public static final class RpcFutureStub extends io.grpc.stub.AbstractStub<RpcFutureStub> {
    private RpcFutureStub(io.grpc.Channel channel) {
      super(channel);
    }

    private RpcFutureStub(io.grpc.Channel channel,
        io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected RpcFutureStub build(io.grpc.Channel channel,
        io.grpc.CallOptions callOptions) {
      return new RpcFutureStub(channel, callOptions);
    }

    /**
     */
    public com.google.common.util.concurrent.ListenableFuture<com.hemayun.bblwheel.RpcOuterClass.Response> call(
        com.hemayun.bblwheel.RpcOuterClass.Request request) {
      return futureUnaryCall(
          getChannel().newCall(METHOD_CALL, getCallOptions()), request);
    }
  }

  private static final int METHODID_CALL = 0;
  private static final int METHODID_CHANNEL = 1;

  private static class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final RpcImplBase serviceImpl;
    private final int methodId;

    public MethodHandlers(RpcImplBase serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_CALL:
          serviceImpl.call((com.hemayun.bblwheel.RpcOuterClass.Request) request,
              (io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Response>) responseObserver);
          break;
        default:
          throw new AssertionError();
      }
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public io.grpc.stub.StreamObserver<Req> invoke(
        io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_CHANNEL:
          return (io.grpc.stub.StreamObserver<Req>) serviceImpl.channel(
              (io.grpc.stub.StreamObserver<com.hemayun.bblwheel.RpcOuterClass.Message>) responseObserver);
        default:
          throw new AssertionError();
      }
    }
  }

  public static io.grpc.ServiceDescriptor getServiceDescriptor() {
    return new io.grpc.ServiceDescriptor(SERVICE_NAME,
        METHOD_CALL,
        METHOD_CHANNEL);
  }

}
