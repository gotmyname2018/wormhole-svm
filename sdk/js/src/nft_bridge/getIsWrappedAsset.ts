import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { getWrappedMeta } from "../solana/nftBridge";

/**
 * Returns whether or not an asset on Solana is a wormhole wrapped asset
 * @param connection
 * @param nftBridgeAddress
 * @param mintAddress
 * @param [commitment]
 * @returns
 */
export async function getIsWrappedAssetSolana(
  connection: Connection,
  nftBridgeAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
) {
  if (!mintAddress) {
    return false;
  }
  return getWrappedMeta(connection, nftBridgeAddress, mintAddress, commitment)
    .catch((_) => null)
    .then((meta) => meta != null);
}

export const getIsWrappedAssetSol = getIsWrappedAssetSolana;
