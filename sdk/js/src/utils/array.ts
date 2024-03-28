import { zeroPad } from "@ethersproject/bytes";
import { PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";

import {
  ChainId,
  ChainName,
  CHAIN_ID_SOLANA,
  CHAIN_ID_UNSET,
  coalesceChainId,
} from "./consts";

export const uint8ArrayToHex = (a: Uint8Array): string =>
  Buffer.from(a).toString("hex");

export const hexToUint8Array = (h: string): Uint8Array => {
  if (h.startsWith("0x")) h = h.slice(2);
  return new Uint8Array(Buffer.from(h, "hex"));
};

/**
 *
 * Convert an address in a wormhole's 32-byte array representation into a chain's
 * native string representation.
 *
 * @throws if address is not the right length for the given chain
 */

export const tryUint8ArrayToNative = (
  a: Uint8Array,
  chain: ChainId | ChainName
): string => {
  const chainId = coalesceChainId(chain);
  if (chainId === CHAIN_ID_SOLANA) {
    return new PublicKey(a).toString();
  } else if (chainId === CHAIN_ID_UNSET) {
    throw Error("uint8ArrayToNative: Chain id unset");
  } else {
    // This case is never reached
    const _: never = chainId;
    throw Error("Don't know how to convert address for chain " + chainId);
  }
};

/**
 *
 * Convert an address in a wormhole's 32-byte hex representation into a chain's native
 * string representation.
 *
 * @throws if address is not the right length for the given chain
 */
export const tryHexToNativeAssetString = (h: string, c: ChainId): string =>
  tryHexToNativeString(h, c);

/**
 *
 * Convert an address in a wormhole's 32-byte hex representation into a chain's native
 * string representation.
 *
 * @throws if address is not the right length for the given chain
 */
export const tryHexToNativeString = (
  h: string,
  c: ChainId | ChainName
): string => tryUint8ArrayToNative(hexToUint8Array(h), c);

/**
 *
 * Convert an address in a chain's native representation into a 32-byte hex string
 * understood by wormhole.
 *
 * @throws if address is a malformed string for the given chain id
 */
export const tryNativeToHexString = (
  address: string,
  chain: ChainId | ChainName
): string => {
  const chainId = coalesceChainId(chain);
  if (chainId === CHAIN_ID_SOLANA) {
    return uint8ArrayToHex(zeroPad(new PublicKey(address).toBytes(), 32));
  } else if (chainId === CHAIN_ID_UNSET) {
    throw Error("nativeToHexString: Chain id unset");
  } else {
    // If this case is reached
    const _: never = chainId;
    throw Error("Don't know how to convert address from chain " + chainId);
  }
};

/**
 *
 * Convert an address in a chain's native representation into a 32-byte array
 * understood by wormhole.
 *
 * @throws if address is a malformed string for the given chain id
 */
export function tryNativeToUint8Array(
  address: string,
  chain: ChainId | ChainName
): Uint8Array {
  const chainId = coalesceChainId(chain);
  return hexToUint8Array(tryNativeToHexString(address, chainId));
}

export function chunks<T>(array: T[], size: number): T[][] {
  return Array.apply<number, T[], T[][]>(
    0,
    new Array(Math.ceil(array.length / size))
  ).map((_, index) => array.slice(index * size, (index + 1) * size));
}

export function textToHexString(name: string): string {
  return Buffer.from(name, "binary").toString("hex");
}

export function textToUint8Array(name: string): Uint8Array {
  return new Uint8Array(Buffer.from(name, "binary"));
}

export function hex(x: string): Buffer {
  return Buffer.from(
    ethers.utils.hexlify(x, { allowMissingPrefix: true }).substring(2),
    "hex"
  );
}

export function ensureHexPrefix(x: string): string {
  return x.substring(0, 2) !== "0x" ? `0x${x}` : x;
}
