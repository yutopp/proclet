import { useMemo } from "react";
import { ServiceType } from "@bufbuild/protobuf";
import { createConnectTransport } from "@connectrpc/connect-web";
import { PromiseClient, createPromiseClient } from "@connectrpc/connect";
import { config } from "./constants";

const transport = createConnectTransport({
  baseUrl: config.BACKEND_URL,
  useBinaryFormat: true,
});

export function useClient<T extends ServiceType>(service: T): PromiseClient<T> {
  return useMemo(() => createPromiseClient(service, transport), [service]);
}
