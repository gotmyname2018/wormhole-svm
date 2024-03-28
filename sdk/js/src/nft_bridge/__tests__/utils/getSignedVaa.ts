import { GetSignedVAAResponse } from "@certusone/wormhole-sdk-proto-web/lib/cjs/publicrpc/v1/publicrpc";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { TransactionResponse } from "@solana/web3.js";
import {
  getEmitterAddressSolana,
  parseSequenceFromLogSolana,
} from "../../../bridge";
import { getSignedVAAWithRetry } from "../../../rpc";
import {
  ChainId,
  CHAIN_ID_SOLANA,
  CONTRACTS,
} from "../../../utils";
import { WORMHOLE_RPC_HOSTS } from "./consts";

export async function getSignedVaaSolana(
  response: TransactionResponse
): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = parseSequenceFromLogSolana(response);
  const emitterAddress = getEmitterAddressSolana(
    CONTRACTS.DEVNET.solana.nft_bridge
  );

  // poll until the guardian(s) witness and sign the vaa
  return getSignedVaa(CHAIN_ID_SOLANA, emitterAddress, sequence);
}

const getSignedVaa = async (
  chain: ChainId,
  emitterAddress: string,
  sequence: string
): Promise<Uint8Array> => {
  const { vaaBytes: signedVAA }: GetSignedVAAResponse =
    await getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      chain,
      emitterAddress,
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );
  return signedVAA;
};
