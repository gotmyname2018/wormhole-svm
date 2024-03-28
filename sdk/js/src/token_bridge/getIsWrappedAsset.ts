import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { getWrappedMeta } from "../solana/tokenBridge";

/**
 * Returns whether or not an asset on Solana is a wormhole wrapped asset
 * @param connection
 * @param tokenBridgeAddress
 * @param mintAddress
 * @param [commitment]
 * @returns
 */
export async function getIsWrappedAssetSolana(
  connection: Connection,
  tokenBridgeAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
): Promise<boolean> {
  if (!mintAddress) {
    return false;
  }
  return getWrappedMeta(connection, tokenBridgeAddress, mintAddress, commitment)
    .catch((_) => null)
    .then((meta) => meta != null);
}

export const getIsWrappedAssetSol = getIsWrappedAssetSolana;
