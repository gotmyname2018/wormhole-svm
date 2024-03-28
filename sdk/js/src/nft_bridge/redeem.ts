import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import {
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCompleteWrappedMetaInstruction,
} from "../solana/nftBridge";
import { CHAIN_ID_SOLANA } from "../utils";
import { parseNftTransferVaa, parseVaa, SignedVaa } from "../vaa";

export async function isNFTVAASolanaNative(
  signedVAA: Uint8Array
): Promise<boolean> {
  return parseVaa(signedVAA).payload.readUInt16BE(33) === CHAIN_ID_SOLANA;
}

export async function redeemOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  nftBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  toAuthorityAddress?: PublicKeyInitData,
  commitment?: Commitment
): Promise<Transaction> {
  const parsed = parseNftTransferVaa(signedVaa);
  const createCompleteTransferInstruction =
    parsed.tokenChain == CHAIN_ID_SOLANA
      ? createCompleteTransferNativeInstruction
      : createCompleteTransferWrappedInstruction;
  const transaction = new Transaction().add(
    createCompleteTransferInstruction(
      nftBridgeAddress,
      bridgeAddress,
      payerAddress,
      parsed,
      toAuthorityAddress
    )
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}

export async function createMetaOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  nftBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  commitment?: Commitment
): Promise<Transaction> {
  const parsed = parseNftTransferVaa(signedVaa);
  if (parsed.tokenChain == CHAIN_ID_SOLANA) {
    return Promise.reject("parsed.tokenChain == CHAIN_ID_SOLANA");
  }
  const transaction = new Transaction().add(
    createCompleteWrappedMetaInstruction(
      nftBridgeAddress,
      bridgeAddress,
      payerAddress,
      parsed
    )
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}
