import {
  ChainId,
  ChainName,
} from "@certusone/wormhole-sdk/lib/esm/utils/consts";

import {
  getEmitterAddressSolana,
} from "@certusone/wormhole-sdk/lib/esm/bridge/getEmitterAddress";

export async function getEmitterAddress(
  chain: ChainId | ChainName,
  addr: string
) {
  if (chain === "solana") {
    // TODO: Create an isSolanaChain()
    addr = getEmitterAddressSolana(addr);
  }
  return addr;
}
