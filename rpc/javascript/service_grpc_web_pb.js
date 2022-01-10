/**
 * @fileoverview gRPC-Web generated client stub for gopm.rpc
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');


var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js')
const proto = {};
proto.gopm = {};
proto.gopm.rpc = require('./service_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.gopm.rpc.GopmClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.gopm.rpc.GopmPromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodDescriptor_Gopm_GetProcessInfo = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/GetProcessInfo',
  grpc.web.MethodType.UNARY,
  google_protobuf_empty_pb.Empty,
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodInfo_Gopm_GetProcessInfo = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.ProcessInfoResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.ProcessInfoResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.getProcessInfo =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/GetProcessInfo',
      request,
      metadata || {},
      methodDescriptor_Gopm_GetProcessInfo,
      callback);
};


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.ProcessInfoResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.getProcessInfo =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/GetProcessInfo',
      request,
      metadata || {},
      methodDescriptor_Gopm_GetProcessInfo);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.StartStopRequest,
 *   !proto.gopm.rpc.StartStopResponse>}
 */
const methodDescriptor_Gopm_StartProcess = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/StartProcess',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.StartStopRequest,
  proto.gopm.rpc.StartStopResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.StartStopResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.StartStopRequest,
 *   !proto.gopm.rpc.StartStopResponse>}
 */
const methodInfo_Gopm_StartProcess = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.StartStopResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.StartStopResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.StartStopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.StartStopResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.StartStopResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.startProcess =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/StartProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_StartProcess,
      callback);
};


/**
 * @param {!proto.gopm.rpc.StartStopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.StartStopResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.startProcess =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/StartProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_StartProcess);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.StartStopRequest,
 *   !proto.gopm.rpc.StartStopResponse>}
 */
const methodDescriptor_Gopm_StopProcess = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/StopProcess',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.StartStopRequest,
  proto.gopm.rpc.StartStopResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.StartStopResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.StartStopRequest,
 *   !proto.gopm.rpc.StartStopResponse>}
 */
const methodInfo_Gopm_StopProcess = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.StartStopResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.StartStopResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.StartStopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.StartStopResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.StartStopResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.stopProcess =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/StopProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_StopProcess,
      callback);
};


/**
 * @param {!proto.gopm.rpc.StartStopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.StartStopResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.stopProcess =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/StopProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_StopProcess);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.StartStopRequest,
 *   !proto.gopm.rpc.StartStopResponse>}
 */
const methodDescriptor_Gopm_RestartProcess = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/RestartProcess',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.StartStopRequest,
  proto.gopm.rpc.StartStopResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.StartStopResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.StartStopRequest,
 *   !proto.gopm.rpc.StartStopResponse>}
 */
const methodInfo_Gopm_RestartProcess = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.StartStopResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.StartStopResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.StartStopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.StartStopResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.StartStopResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.restartProcess =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/RestartProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_RestartProcess,
      callback);
};


/**
 * @param {!proto.gopm.rpc.StartStopRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.StartStopResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.restartProcess =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/RestartProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_RestartProcess);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.StartStopAllRequest,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodDescriptor_Gopm_StartAllProcesses = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/StartAllProcesses',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.StartStopAllRequest,
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopAllRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.StartStopAllRequest,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodInfo_Gopm_StartAllProcesses = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopAllRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.StartStopAllRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.ProcessInfoResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.ProcessInfoResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.startAllProcesses =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/StartAllProcesses',
      request,
      metadata || {},
      methodDescriptor_Gopm_StartAllProcesses,
      callback);
};


/**
 * @param {!proto.gopm.rpc.StartStopAllRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.ProcessInfoResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.startAllProcesses =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/StartAllProcesses',
      request,
      metadata || {},
      methodDescriptor_Gopm_StartAllProcesses);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.StartStopAllRequest,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodDescriptor_Gopm_StopAllProcesses = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/StopAllProcesses',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.StartStopAllRequest,
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopAllRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.StartStopAllRequest,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodInfo_Gopm_StopAllProcesses = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.gopm.rpc.StartStopAllRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.StartStopAllRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.ProcessInfoResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.ProcessInfoResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.stopAllProcesses =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/StopAllProcesses',
      request,
      metadata || {},
      methodDescriptor_Gopm_StopAllProcesses,
      callback);
};


/**
 * @param {!proto.gopm.rpc.StartStopAllRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.ProcessInfoResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.stopAllProcesses =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/StopAllProcesses',
      request,
      metadata || {},
      methodDescriptor_Gopm_StopAllProcesses);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.google.protobuf.Empty,
 *   !proto.google.protobuf.Empty>}
 */
const methodDescriptor_Gopm_Shutdown = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/Shutdown',
  grpc.web.MethodType.UNARY,
  google_protobuf_empty_pb.Empty,
  google_protobuf_empty_pb.Empty,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  google_protobuf_empty_pb.Empty.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.google.protobuf.Empty,
 *   !proto.google.protobuf.Empty>}
 */
const methodInfo_Gopm_Shutdown = new grpc.web.AbstractClientBase.MethodInfo(
  google_protobuf_empty_pb.Empty,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  google_protobuf_empty_pb.Empty.deserializeBinary
);


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.google.protobuf.Empty)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.google.protobuf.Empty>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.shutdown =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/Shutdown',
      request,
      metadata || {},
      methodDescriptor_Gopm_Shutdown,
      callback);
};


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.google.protobuf.Empty>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.shutdown =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/Shutdown',
      request,
      metadata || {},
      methodDescriptor_Gopm_Shutdown);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.ReloadConfigResponse>}
 */
const methodDescriptor_Gopm_ReloadConfig = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/ReloadConfig',
  grpc.web.MethodType.UNARY,
  google_protobuf_empty_pb.Empty,
  proto.gopm.rpc.ReloadConfigResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ReloadConfigResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.ReloadConfigResponse>}
 */
const methodInfo_Gopm_ReloadConfig = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.ReloadConfigResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ReloadConfigResponse.deserializeBinary
);


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.ReloadConfigResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.ReloadConfigResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.reloadConfig =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/ReloadConfig',
      request,
      metadata || {},
      methodDescriptor_Gopm_ReloadConfig,
      callback);
};


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.ReloadConfigResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.reloadConfig =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/ReloadConfig',
      request,
      metadata || {},
      methodDescriptor_Gopm_ReloadConfig);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.TailLogRequest,
 *   !proto.gopm.rpc.TailLogResponse>}
 */
const methodDescriptor_Gopm_TailLog = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/TailLog',
  grpc.web.MethodType.SERVER_STREAMING,
  proto.gopm.rpc.TailLogRequest,
  proto.gopm.rpc.TailLogResponse,
  /**
   * @param {!proto.gopm.rpc.TailLogRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.TailLogResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.TailLogRequest,
 *   !proto.gopm.rpc.TailLogResponse>}
 */
const methodInfo_Gopm_TailLog = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.TailLogResponse,
  /**
   * @param {!proto.gopm.rpc.TailLogRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.TailLogResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.TailLogRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.TailLogResponse>}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.tailLog =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/gopm.rpc.Gopm/TailLog',
      request,
      metadata || {},
      methodDescriptor_Gopm_TailLog);
};


/**
 * @param {!proto.gopm.rpc.TailLogRequest} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.TailLogResponse>}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmPromiseClient.prototype.tailLog =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/gopm.rpc.Gopm/TailLog',
      request,
      metadata || {},
      methodDescriptor_Gopm_TailLog);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.SignalProcessRequest,
 *   !proto.google.protobuf.Empty>}
 */
const methodDescriptor_Gopm_SignalProcess = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/SignalProcess',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.SignalProcessRequest,
  google_protobuf_empty_pb.Empty,
  /**
   * @param {!proto.gopm.rpc.SignalProcessRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  google_protobuf_empty_pb.Empty.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.SignalProcessRequest,
 *   !proto.google.protobuf.Empty>}
 */
const methodInfo_Gopm_SignalProcess = new grpc.web.AbstractClientBase.MethodInfo(
  google_protobuf_empty_pb.Empty,
  /**
   * @param {!proto.gopm.rpc.SignalProcessRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  google_protobuf_empty_pb.Empty.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.SignalProcessRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.google.protobuf.Empty)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.google.protobuf.Empty>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.signalProcess =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/SignalProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_SignalProcess,
      callback);
};


/**
 * @param {!proto.gopm.rpc.SignalProcessRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.google.protobuf.Empty>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.signalProcess =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/SignalProcess',
      request,
      metadata || {},
      methodDescriptor_Gopm_SignalProcess);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopm.rpc.SignalProcessRequest,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodDescriptor_Gopm_SignalAllProcesses = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/SignalAllProcesses',
  grpc.web.MethodType.UNARY,
  proto.gopm.rpc.SignalProcessRequest,
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.gopm.rpc.SignalProcessRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.gopm.rpc.SignalProcessRequest,
 *   !proto.gopm.rpc.ProcessInfoResponse>}
 */
const methodInfo_Gopm_SignalAllProcesses = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.ProcessInfoResponse,
  /**
   * @param {!proto.gopm.rpc.SignalProcessRequest} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.ProcessInfoResponse.deserializeBinary
);


/**
 * @param {!proto.gopm.rpc.SignalProcessRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.ProcessInfoResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.ProcessInfoResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.signalAllProcesses =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/SignalAllProcesses',
      request,
      metadata || {},
      methodDescriptor_Gopm_SignalAllProcesses,
      callback);
};


/**
 * @param {!proto.gopm.rpc.SignalProcessRequest} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.ProcessInfoResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.signalAllProcesses =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/SignalAllProcesses',
      request,
      metadata || {},
      methodDescriptor_Gopm_SignalAllProcesses);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.DumpConfigResponse>}
 */
const methodDescriptor_Gopm_DumpConfig = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/DumpConfig',
  grpc.web.MethodType.UNARY,
  google_protobuf_empty_pb.Empty,
  proto.gopm.rpc.DumpConfigResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.DumpConfigResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.DumpConfigResponse>}
 */
const methodInfo_Gopm_DumpConfig = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.DumpConfigResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.DumpConfigResponse.deserializeBinary
);


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.gopm.rpc.DumpConfigResponse)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.DumpConfigResponse>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.dumpConfig =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopm.rpc.Gopm/DumpConfig',
      request,
      metadata || {},
      methodDescriptor_Gopm_DumpConfig,
      callback);
};


/**
 * @param {!proto.google.protobuf.Empty} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopm.rpc.DumpConfigResponse>}
 *     Promise that resolves to the response
 */
proto.gopm.rpc.GopmPromiseClient.prototype.dumpConfig =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopm.rpc.Gopm/DumpConfig',
      request,
      metadata || {},
      methodDescriptor_Gopm_DumpConfig);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.WatchStateResponse>}
 */
const methodDescriptor_Gopm_WatchState = new grpc.web.MethodDescriptor(
  '/gopm.rpc.Gopm/WatchState',
  grpc.web.MethodType.SERVER_STREAMING,
  google_protobuf_empty_pb.Empty,
  proto.gopm.rpc.WatchStateResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.WatchStateResponse.deserializeBinary
);


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.google.protobuf.Empty,
 *   !proto.gopm.rpc.WatchStateResponse>}
 */
const methodInfo_Gopm_WatchState = new grpc.web.AbstractClientBase.MethodInfo(
  proto.gopm.rpc.WatchStateResponse,
  /**
   * @param {!proto.google.protobuf.Empty} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopm.rpc.WatchStateResponse.deserializeBinary
);


/**
 * @param {!proto.google.protobuf.Empty} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.WatchStateResponse>}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmClient.prototype.watchState =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/gopm.rpc.Gopm/WatchState',
      request,
      metadata || {},
      methodDescriptor_Gopm_WatchState);
};


/**
 * @param {!proto.google.protobuf.Empty} request The request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.gopm.rpc.WatchStateResponse>}
 *     The XHR Node Readable Stream
 */
proto.gopm.rpc.GopmPromiseClient.prototype.watchState =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/gopm.rpc.Gopm/WatchState',
      request,
      metadata || {},
      methodDescriptor_Gopm_WatchState);
};


module.exports = proto.gopm.rpc;

