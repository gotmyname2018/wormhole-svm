import {
  CHAIN_ID_SOLANA,
  ChainId,
  ChainName,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { Connection as SolanaConnection } from "@solana/web3.js";
import { NETWORKS } from "../../consts";
import { Network } from "../../utils";
import { impossible } from "../../vaa";

export type ChainProvider<T extends ChainId | ChainName> = T extends
  | "solana" | typeof CHAIN_ID_SOLANA
  ? SolanaConnection
  : never;

export const getProviderForChain = <T extends ChainId | ChainName>(
  chain: T,
  network: Network,
  options?: { rpc?: string; [opt: string]: any }
): ChainProvider<T> => {
  const chainName = coalesceChainName(chain);
  const rpc = options?.rpc ?? NETWORKS[network][chainName].rpc;
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${chainName}`);
  }

  switch (chainName) {
    case "unset":
      throw new Error("Chain not set");
    case "solana":
      return new SolanaConnection(rpc, "confirmed") as ChainProvider<T>;
    default:
      impossible(chainName);
  }
};
