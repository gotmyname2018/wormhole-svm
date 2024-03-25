import {
  WormholeWrappedInfo,
  getOriginalAssetSolana,
} from "@certusone/wormhole-sdk/lib/esm/token_bridge/getOriginalAsset";
import {
  ChainId,
  ChainName,
  coalesceChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { CONTRACTS } from "../../consts";
import { Network } from "../../utils";
import { impossible } from "../../vaa";
import { getProviderForChain } from "./provider";

export const getOriginalAsset = async (
  chain: ChainId | ChainName,
  network: Network,
  assetAddress: string,
  rpc?: string
): Promise<WormholeWrappedInfo> => {
  const chainName = coalesceChainName(chain);
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
      return getOriginalAssetSolana(provider, tokenBridgeAddress, assetAddress);
    }
    default:
      impossible(chainName);
  }
};
