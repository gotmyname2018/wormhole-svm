import {
  getForeignAssetSolana,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/getForeignAsset";
import { tryNativeToUint8Array } from "@certusone/wormhole-sdk/lib/esm/utils/array";
import {
  ChainId,
  ChainName,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACTS } from "../../consts";
import { Network } from "../../utils";
import { impossible } from "../../vaa";
import { getProviderForChain } from "./provider";

export const getWrappedAssetAddress = async (
  chain: ChainId | ChainName,
  network: Network,
  originChain: ChainId | ChainName,
  originAddress: string,
  rpc?: string
): Promise<string | null> => {
  const chainName = coalesceChainName(chain);
  const originAddressUint8Array = tryNativeToUint8Array(
    originAddress,
    originChain
  );
  const tokenBridgeAddress = CONTRACTS[network][chainName].token_bridge;
  if (!tokenBridgeAddress) {
    throw new Error(
      `Token bridge address not defined for ${chainName} ${network}`
    );
  }

  switch (chainName) {
    case "unset":
      throw new Error("Chain not set");
    case "solana": {
      const provider = getProviderForChain(chainName, network, { rpc });
      return getForeignAssetSolana(
        provider,
        tokenBridgeAddress,
        originChain,
        originAddressUint8Array
      );
    }
    default:
      impossible(chainName);
  }
};
