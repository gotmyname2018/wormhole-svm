import { PublicKeyInitData } from "@solana/web3.js";
import { deriveWormholeEmitterKey } from "../solana/wormhole";

export function getEmitterAddressSolana(programAddress: PublicKeyInitData) {
  return deriveWormholeEmitterKey(programAddress).toBuffer().toString("hex");
}
