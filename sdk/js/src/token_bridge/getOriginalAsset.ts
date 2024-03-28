import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
} from "@solana/web3.js";
import { getWrappedMeta } from "../solana/tokenBridge";
import {
  CHAIN_ID_SOLANA,
  ChainId,
} from "../utils";

// TODO: remove `as ChainId` and return number in next minor version as we can't ensure it will match our type definition
export interface WormholeWrappedInfo {
  isWrapped: boolean;
  chainId: ChainId;
  assetAddress: Uint8Array;
}

/**
 * Returns a origin chain and asset address on {originChain} for a provided Wormhole wrapped address
 * @param connection
 * @param tokenBridgeAddress
 * @param mintAddress
 * @param [commitment]
 * @returns
 */
export async function getOriginalAssetSolana(
  connection: Connection,
  tokenBridgeAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
): Promise<WormholeWrappedInfo> {
  try {
    const mint = new PublicKey(mintAddress);

    return getWrappedMeta(
      connection,
      tokenBridgeAddress,
      mintAddress,
      commitment
    )
      .catch((_) => null)
      .then((meta) => {
        if (meta === null) {
          return {
            isWrapped: false,
            chainId: CHAIN_ID_SOLANA,
            assetAddress: mint.toBytes(),
          };
        } else {
          return {
            isWrapped: true,
            chainId: meta.chain as ChainId,
            assetAddress: Uint8Array.from(meta.tokenAddress),
          };
        }
      });
  } catch (_) {
    return {
      isWrapped: false,
      chainId: CHAIN_ID_SOLANA,
      assetAddress: new Uint8Array(32),
    };
  }
}

export const getOriginalAssetSol = getOriginalAssetSolana;
