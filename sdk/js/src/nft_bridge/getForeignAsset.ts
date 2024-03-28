import { BN } from "@project-serum/anchor";
import { PublicKeyInitData } from "@solana/web3.js";
import { isBytes } from "ethers/lib/utils";
import { deriveWrappedMintKey } from "../solana/nftBridge";
import {
  ChainId,
  ChainName,
  coalesceChainId,
} from "../utils";

/**
 * Returns a foreign asset address on Solana for a provided native chain and asset address
 * @param nftBridgeAddress
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @returns
 */
export async function getForeignAssetSolana(
  nftBridgeAddress: PublicKeyInitData,
  originChain: ChainId | ChainName,
  originAsset: string | Uint8Array | Buffer,
  tokenId: Uint8Array | Buffer | bigint
): Promise<string> {
  // we don't require NFT accounts to exist, so don't check them.
  return deriveWrappedMintKey(
    nftBridgeAddress,
    coalesceChainId(originChain) as number,
    originAsset,
    isBytes(tokenId) ? BigInt(new BN(tokenId).toString()) : tokenId
  ).toString();
}

export const getForeignAssetSol = getForeignAssetSolana;
