import * as grpcWeb from 'grpc-web';

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb';
import * as service_pb from './service_pb';


export class GopmClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getProcessInfo(
    request: google_protobuf_empty_pb.Empty,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.ProcessInfoResponse>;

  startProcess(
    request: service_pb.StartStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.StartStopResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.StartStopResponse>;

  stopProcess(
    request: service_pb.StartStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.StartStopResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.StartStopResponse>;

  restartProcess(
    request: service_pb.StartStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.StartStopResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.StartStopResponse>;

  startAllProcesses(
    request: service_pb.StartStopAllRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.ProcessInfoResponse>;

  stopAllProcesses(
    request: service_pb.StartStopAllRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.ProcessInfoResponse>;

  shutdown(
    request: google_protobuf_empty_pb.Empty,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: google_protobuf_empty_pb.Empty) => void
  ): grpcWeb.ClientReadableStream<google_protobuf_empty_pb.Empty>;

  reloadConfig(
    request: google_protobuf_empty_pb.Empty,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.ReloadConfigResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.ReloadConfigResponse>;

  tailLog(
    request: service_pb.TailLogRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<service_pb.TailLogResponse>;

  signalProcess(
    request: service_pb.SignalProcessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: google_protobuf_empty_pb.Empty) => void
  ): grpcWeb.ClientReadableStream<google_protobuf_empty_pb.Empty>;

  signalAllProcesses(
    request: service_pb.SignalProcessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.ProcessInfoResponse>;

  dumpConfig(
    request: google_protobuf_empty_pb.Empty,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: service_pb.DumpConfigResponse) => void
  ): grpcWeb.ClientReadableStream<service_pb.DumpConfigResponse>;

  watchState(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<service_pb.WatchStateResponse>;

}

export class GopmPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: any; });

  getProcessInfo(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.ProcessInfoResponse>;

  startProcess(
    request: service_pb.StartStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.StartStopResponse>;

  stopProcess(
    request: service_pb.StartStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.StartStopResponse>;

  restartProcess(
    request: service_pb.StartStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.StartStopResponse>;

  startAllProcesses(
    request: service_pb.StartStopAllRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.ProcessInfoResponse>;

  stopAllProcesses(
    request: service_pb.StartStopAllRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.ProcessInfoResponse>;

  shutdown(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<google_protobuf_empty_pb.Empty>;

  reloadConfig(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.ReloadConfigResponse>;

  tailLog(
    request: service_pb.TailLogRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<service_pb.TailLogResponse>;

  signalProcess(
    request: service_pb.SignalProcessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<google_protobuf_empty_pb.Empty>;

  signalAllProcesses(
    request: service_pb.SignalProcessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.ProcessInfoResponse>;

  dumpConfig(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<service_pb.DumpConfigResponse>;

  watchState(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<service_pb.WatchStateResponse>;

}

