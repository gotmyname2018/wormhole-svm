import {
  TransactionResponse,
  VersionedTransactionResponse,
} from "@solana/web3.js";

const SOLANA_SEQ_LOG = "Program log: Sequence: ";
export function parseSequenceFromLogSolana(
  info: TransactionResponse | VersionedTransactionResponse
) {
  // TODO: better parsing, safer
  const sequence = info.meta?.logMessages
    ?.filter((msg) => msg.startsWith(SOLANA_SEQ_LOG))?.[0]
    ?.replace(SOLANA_SEQ_LOG, "");
  if (!sequence) {
    throw new Error("sequence not found");
  }
  return sequence.toString();
}

export function parseSequencesFromLogSolana(info: TransactionResponse) {
  // TODO: better parsing, safer
  return info.meta?.logMessages
    ?.filter((msg) => msg.startsWith(SOLANA_SEQ_LOG))
    .map((msg) => msg.replace(SOLANA_SEQ_LOG, ""));
}
