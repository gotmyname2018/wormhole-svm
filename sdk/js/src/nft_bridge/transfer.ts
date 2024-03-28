import { BN } from "@project-serum/anchor";
import {
  Commitment,
  Connection,
  Keypair,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import { isBytes } from "ethers/lib/utils";
import { createBridgeFeeTransferInstruction } from "../solana";
import {
  createApproveAuthoritySignerInstruction,
  createTransferNativeInstruction,
  createTransferWrappedInstruction,
} from "../solana/nftBridge";
import {
  ChainId,
  ChainName,
  CHAIN_ID_SOLANA,
  coalesceChainId,
  createNonce,
} from "../utils";

export async function transferFromSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  nftBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  fromAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  targetAddress: Uint8Array | Buffer,
  targetChain: ChainId | ChainName,
  originAddress?: Uint8Array | Buffer,
  originChain?: ChainId | ChainName,
  originTokenId?: Uint8Array | Buffer | number | bigint,
  commitment?: Commitment
): Promise<Transaction> {
  const originChainId: ChainId | undefined = originChain
    ? coalesceChainId(originChain)
    : undefined;
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await createBridgeFeeTransferInstruction(
    connection,
    bridgeAddress,
    payerAddress
  );
  const approvalIx = createApproveAuthoritySignerInstruction(
    nftBridgeAddress,
    fromAddress,
    payerAddress
  );
  let message = Keypair.generate();
  const isSolanaNative =
    originChain === undefined || originChain === CHAIN_ID_SOLANA;
  if (!isSolanaNative && (!originAddress || !originTokenId)) {
    return Promise.reject(
      "originAddress and originTokenId are required when specifying originChain"
    );
  }
  const nftBridgeTransferIx = isSolanaNative
    ? createTransferNativeInstruction(
        nftBridgeAddress,
        bridgeAddress,
        payerAddress,
        message.publicKey,
        fromAddress,
        mintAddress,
        nonce,
        targetAddress,
        coalesceChainId(targetChain)
      )
    : createTransferWrappedInstruction(
        nftBridgeAddress,
        bridgeAddress,
        payerAddress,
        message.publicKey,
        fromAddress,
        payerAddress,
        originChainId!,
        originAddress!,
        isBytes(originTokenId)
          ? BigInt(new BN(originTokenId).toString())
          : originTokenId!,
        nonce,
        targetAddress,
        coalesceChainId(targetChain)
      );
  const transaction = new Transaction().add(
    transferIx,
    approvalIx,
    nftBridgeTransferIx
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(message);
  return transaction;
}
