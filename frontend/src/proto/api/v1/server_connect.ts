// @generated by protoc-gen-connect-es v1.2.0 with parameter "target=ts"
// @generated from file proto/api/v1/server.proto (package v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { RunOneshotRequest, RunOneshotResponse } from "./server_pb.js";
import { MethodKind } from "@bufbuild/protobuf";

/**
 * @generated from service v1.KoyaService
 */
export const KoyaService = {
  typeName: "v1.KoyaService",
  methods: {
    /**
     * @generated from rpc v1.KoyaService.RunOneshot
     */
    runOneshot: {
      name: "RunOneshot",
      I: RunOneshotRequest,
      O: RunOneshotResponse,
      kind: MethodKind.ServerStreaming,
    },
  }
} as const;

