import { Commitment, Connection, PublicKeyInitData } from "@solana/web3.js";
import { getClaim } from "../solana/wormhole";
import { SignedVaa, parseVaa } from "../vaa/wormhole";

export async function getIsTransferCompletedSolana(
  tokenBridgeAddress: PublicKeyInitData,
  signedVAA: SignedVaa,
  connection: Connection,
  commitment?: Commitment
): Promise<boolean> {
  const parsed = parseVaa(signedVAA);
  return getClaim(
    connection,
    tokenBridgeAddress,
    parsed.emitterAddress,
    parsed.emitterChain,
    parsed.sequence,
    commitment
  ).catch((e) => false);
}
