import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { expect } from "@jest/globals";
import { ChainId, getSignedVAAWithRetry } from "../../..";
import {
  WORMHOLE_RPC_HOSTS,
} from "./consts";

export async function getSignedVAABySequence(
  chainId: ChainId,
  sequence: string,
  emitterAddress: string
): Promise<Uint8Array> {
  //Note, if handed a sequence which doesn't exist or was skipped for consensus this will retry until the timeout.
  const { vaaBytes } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    chainId,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(), //This should only be needed when running in node.
    }
  );

  return vaaBytes;
}

// https://github.com/microsoft/TypeScript/issues/34523
export const assertIsNotNull: <T>(x: T | null) => asserts x is T = (x) => {
  expect(x).not.toBeNull();
};

export const assertIsNotNullOrUndefined: <T>(
  x: T | null | undefined
) => asserts x is T = (x) => {
  expect(x).not.toBeNull();
  expect(x).not.toBeUndefined();
};
