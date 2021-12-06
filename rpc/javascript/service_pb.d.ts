import * as jspb from 'google-protobuf'

import * as google_protobuf_empty_pb from 'google-protobuf/google/protobuf/empty_pb';


export class VersionResponse extends jspb.Message {
  getVersion(): string;
  setVersion(value: string): VersionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): VersionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: VersionResponse): VersionResponse.AsObject;
  static serializeBinaryToWriter(message: VersionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): VersionResponse;
  static deserializeBinaryFromReader(message: VersionResponse, reader: jspb.BinaryReader): VersionResponse;
}

export namespace VersionResponse {
  export type AsObject = {
    version: string,
  }
}

export class ProcessInfoResponse extends jspb.Message {
  getProcessesList(): Array<ProcessInfo>;
  setProcessesList(value: Array<ProcessInfo>): ProcessInfoResponse;
  clearProcessesList(): ProcessInfoResponse;
  addProcesses(value?: ProcessInfo, index?: number): ProcessInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProcessInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ProcessInfoResponse): ProcessInfoResponse.AsObject;
  static serializeBinaryToWriter(message: ProcessInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProcessInfoResponse;
  static deserializeBinaryFromReader(message: ProcessInfoResponse, reader: jspb.BinaryReader): ProcessInfoResponse;
}

export namespace ProcessInfoResponse {
  export type AsObject = {
    processesList: Array<ProcessInfo.AsObject>,
  }
}

export class ProcessInfo extends jspb.Message {
  getName(): string;
  setName(value: string): ProcessInfo;

  getGroup(): string;
  setGroup(value: string): ProcessInfo;

  getDescription(): string;
  setDescription(value: string): ProcessInfo;

  getStart(): number;
  setStart(value: number): ProcessInfo;

  getStop(): number;
  setStop(value: number): ProcessInfo;

  getNow(): number;
  setNow(value: number): ProcessInfo;

  getState(): string;
  setState(value: string): ProcessInfo;

  getSpawnErr(): string;
  setSpawnErr(value: string): ProcessInfo;

  getExitStatus(): number;
  setExitStatus(value: number): ProcessInfo;

  getLogfile(): string;
  setLogfile(value: string): ProcessInfo;

  getPid(): number;
  setPid(value: number): ProcessInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProcessInfo.AsObject;
  static toObject(includeInstance: boolean, msg: ProcessInfo): ProcessInfo.AsObject;
  static serializeBinaryToWriter(message: ProcessInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProcessInfo;
  static deserializeBinaryFromReader(message: ProcessInfo, reader: jspb.BinaryReader): ProcessInfo;
}

export namespace ProcessInfo {
  export type AsObject = {
    name: string,
    group: string,
    description: string,
    start: number,
    stop: number,
    now: number,
    state: string,
    spawnErr: string,
    exitStatus: number,
    logfile: string,
    pid: number,
  }
}

export class StartStopRequest extends jspb.Message {
  getName(): string;
  setName(value: string): StartStopRequest;

  getWait(): boolean;
  setWait(value: boolean): StartStopRequest;

  getLabelsMap(): jspb.Map<string, string>;
  clearLabelsMap(): StartStopRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartStopRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartStopRequest): StartStopRequest.AsObject;
  static serializeBinaryToWriter(message: StartStopRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartStopRequest;
  static deserializeBinaryFromReader(message: StartStopRequest, reader: jspb.BinaryReader): StartStopRequest;
}

export namespace StartStopRequest {
  export type AsObject = {
    name: string,
    wait: boolean,
    labelsMap: Array<[string, string]>,
  }
}

export class StartStopResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartStopResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StartStopResponse): StartStopResponse.AsObject;
  static serializeBinaryToWriter(message: StartStopResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartStopResponse;
  static deserializeBinaryFromReader(message: StartStopResponse, reader: jspb.BinaryReader): StartStopResponse;
}

export namespace StartStopResponse {
  export type AsObject = {
  }
}

export class StartStopAllRequest extends jspb.Message {
  getWait(): boolean;
  setWait(value: boolean): StartStopAllRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartStopAllRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartStopAllRequest): StartStopAllRequest.AsObject;
  static serializeBinaryToWriter(message: StartStopAllRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartStopAllRequest;
  static deserializeBinaryFromReader(message: StartStopAllRequest, reader: jspb.BinaryReader): StartStopAllRequest;
}

export namespace StartStopAllRequest {
  export type AsObject = {
    wait: boolean,
  }
}

export class ReloadConfigResponse extends jspb.Message {
  getAddedGroupList(): Array<string>;
  setAddedGroupList(value: Array<string>): ReloadConfigResponse;
  clearAddedGroupList(): ReloadConfigResponse;
  addAddedGroup(value: string, index?: number): ReloadConfigResponse;

  getChangedGroupList(): Array<string>;
  setChangedGroupList(value: Array<string>): ReloadConfigResponse;
  clearChangedGroupList(): ReloadConfigResponse;
  addChangedGroup(value: string, index?: number): ReloadConfigResponse;

  getRemovedGroupList(): Array<string>;
  setRemovedGroupList(value: Array<string>): ReloadConfigResponse;
  clearRemovedGroupList(): ReloadConfigResponse;
  addRemovedGroup(value: string, index?: number): ReloadConfigResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReloadConfigResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ReloadConfigResponse): ReloadConfigResponse.AsObject;
  static serializeBinaryToWriter(message: ReloadConfigResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReloadConfigResponse;
  static deserializeBinaryFromReader(message: ReloadConfigResponse, reader: jspb.BinaryReader): ReloadConfigResponse;
}

export namespace ReloadConfigResponse {
  export type AsObject = {
    addedGroupList: Array<string>,
    changedGroupList: Array<string>,
    removedGroupList: Array<string>,
  }
}

export class TailLogRequest extends jspb.Message {
  getName(): string;
  setName(value: string): TailLogRequest;

  getBackloglines(): number;
  setBackloglines(value: number): TailLogRequest;

  getNofollow(): boolean;
  setNofollow(value: boolean): TailLogRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TailLogRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TailLogRequest): TailLogRequest.AsObject;
  static serializeBinaryToWriter(message: TailLogRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TailLogRequest;
  static deserializeBinaryFromReader(message: TailLogRequest, reader: jspb.BinaryReader): TailLogRequest;
}

export namespace TailLogRequest {
  export type AsObject = {
    name: string,
    backloglines: number,
    nofollow: boolean,
  }
}

export class TailLogResponse extends jspb.Message {
  getLines(): Uint8Array | string;
  getLines_asU8(): Uint8Array;
  getLines_asB64(): string;
  setLines(value: Uint8Array | string): TailLogResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TailLogResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TailLogResponse): TailLogResponse.AsObject;
  static serializeBinaryToWriter(message: TailLogResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TailLogResponse;
  static deserializeBinaryFromReader(message: TailLogResponse, reader: jspb.BinaryReader): TailLogResponse;
}

export namespace TailLogResponse {
  export type AsObject = {
    lines: Uint8Array | string,
  }
}

export class SignalProcessRequest extends jspb.Message {
  getName(): string;
  setName(value: string): SignalProcessRequest;

  getSignal(): ProcessSignal;
  setSignal(value: ProcessSignal): SignalProcessRequest;

  getLabelsMap(): jspb.Map<string, string>;
  clearLabelsMap(): SignalProcessRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SignalProcessRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SignalProcessRequest): SignalProcessRequest.AsObject;
  static serializeBinaryToWriter(message: SignalProcessRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SignalProcessRequest;
  static deserializeBinaryFromReader(message: SignalProcessRequest, reader: jspb.BinaryReader): SignalProcessRequest;
}

export namespace SignalProcessRequest {
  export type AsObject = {
    name: string,
    signal: ProcessSignal,
    labelsMap: Array<[string, string]>,
  }
}

export enum ProcessSignal { 
  HUP = 0,
  INT = 1,
  QUIT = 2,
  KILL = 3,
  USR1 = 4,
  USR2 = 5,
  TERM = 6,
  STOP = 7,
  CONT = 8,
}
