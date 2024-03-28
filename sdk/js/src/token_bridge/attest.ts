import {
  Commitment,
  Connection,
  Keypair,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import { createBridgeFeeTransferInstruction } from "../solana";
import { createAttestTokenInstruction } from "../solana/tokenBridge";
import { createNonce } from "../utils/createNonce";

export async function attestFromSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  commitment?: Commitment
): Promise<Transaction> {
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await createBridgeFeeTransferInstruction(
    connection,
    bridgeAddress,
    payerAddress
  );
  const messageKey = Keypair.generate();
  const attestIx = createAttestTokenInstruction(
    tokenBridgeAddress,
    bridgeAddress,
    payerAddress,
    mintAddress,
    messageKey.publicKey,
    nonce
  );
  const transaction = new Transaction().add(transferIx, attestIx);
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(messageKey);
  return transaction;
}
