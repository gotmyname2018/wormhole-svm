import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { deriveWrappedMintKey, getWrappedMeta } from "../solana/tokenBridge";
import {
  ChainId,
  ChainName,
  coalesceChainId,
} from "../utils";

/**
 * Returns a foreign asset address on Solana for a provided native chain and asset address
 * @param connection
 * @param tokenBridgeAddress
 * @param originChain
 * @param originAsset zero pad to 32 bytes
 * @param [commitment]
 * @returns
 */
export async function getForeignAssetSolana(
  connection: Connection,
  tokenBridgeAddress: PublicKeyInitData,
  originChain: ChainId | ChainName,
  originAsset: Uint8Array,
  commitment?: Commitment
): Promise<string | null> {
  const mint = deriveWrappedMintKey(
    tokenBridgeAddress,
    coalesceChainId(originChain) as number,
    originAsset
  );
  return getWrappedMeta(connection, tokenBridgeAddress, mint, commitment)
    .catch((_) => null)
    .then((meta) => (meta === null ? null : mint.toString()));
}
