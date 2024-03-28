import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { getClaim } from "../solana/wormhole";
import { parseVaa, SignedVaa } from "../vaa/wormhole";

export async function getIsTransferCompletedSolana(
  nftBridgeAddress: PublicKeyInitData,
  signedVAA: SignedVaa,
  connection: Connection,
  commitment?: Commitment
) {
  const parsed = parseVaa(signedVAA);
  return getClaim(
    connection,
    nftBridgeAddress,
    parsed.emitterAddress,
    parsed.emitterChain,
    parsed.sequence,
    commitment
  ).catch((e) => false);
}
