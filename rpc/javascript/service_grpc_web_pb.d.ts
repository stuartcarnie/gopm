import * as grpcWeb from 'grpc-web';

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb';

import {
  ProcessInfoResponse,
  ReloadConfigResponse,
  SignalProcessRequest,
  StartStopAllRequest,
  StartStopRequest,
  StartStopResponse,
  TailLogRequest,
  TailLogResponse,
  VersionResponse} from './service_pb';

export class GopmClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: string; });

  getVersion(
    request: google_protobuf_empty_pb.Empty,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: VersionResponse) => void
  ): grpcWeb.ClientReadableStream<VersionResponse>;

  getProcessInfo(
    request: google_protobuf_empty_pb.Empty,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<ProcessInfoResponse>;

  startProcess(
    request: StartStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: StartStopResponse) => void
  ): grpcWeb.ClientReadableStream<StartStopResponse>;

  stopProcess(
    request: StartStopRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: StartStopResponse) => void
  ): grpcWeb.ClientReadableStream<StartStopResponse>;

  startAllProcesses(
    request: StartStopAllRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<ProcessInfoResponse>;

  stopAllProcesses(
    request: StartStopAllRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<ProcessInfoResponse>;

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
               response: ReloadConfigResponse) => void
  ): grpcWeb.ClientReadableStream<ReloadConfigResponse>;

  tailLog(
    request: TailLogRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<TailLogResponse>;

  signalProcess(
    request: SignalProcessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: google_protobuf_empty_pb.Empty) => void
  ): grpcWeb.ClientReadableStream<google_protobuf_empty_pb.Empty>;

  signalProcessGroup(
    request: SignalProcessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<ProcessInfoResponse>;

  signalAllProcesses(
    request: SignalProcessRequest,
    metadata: grpcWeb.Metadata | undefined,
    callback: (err: grpcWeb.Error,
               response: ProcessInfoResponse) => void
  ): grpcWeb.ClientReadableStream<ProcessInfoResponse>;

}

export class GopmPromiseClient {
  constructor (hostname: string,
               credentials?: null | { [index: string]: string; },
               options?: null | { [index: string]: string; });

  getVersion(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<VersionResponse>;

  getProcessInfo(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<ProcessInfoResponse>;

  startProcess(
    request: StartStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<StartStopResponse>;

  stopProcess(
    request: StartStopRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<StartStopResponse>;

  startAllProcesses(
    request: StartStopAllRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<ProcessInfoResponse>;

  stopAllProcesses(
    request: StartStopAllRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<ProcessInfoResponse>;

  shutdown(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<google_protobuf_empty_pb.Empty>;

  reloadConfig(
    request: google_protobuf_empty_pb.Empty,
    metadata?: grpcWeb.Metadata
  ): Promise<ReloadConfigResponse>;

  tailLog(
    request: TailLogRequest,
    metadata?: grpcWeb.Metadata
  ): grpcWeb.ClientReadableStream<TailLogResponse>;

  signalProcess(
    request: SignalProcessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<google_protobuf_empty_pb.Empty>;

  signalProcessGroup(
    request: SignalProcessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<ProcessInfoResponse>;

  signalAllProcesses(
    request: SignalProcessRequest,
    metadata?: grpcWeb.Metadata
  ): Promise<ProcessInfoResponse>;

}

