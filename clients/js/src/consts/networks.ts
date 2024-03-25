import { ChainName } from "@certusone/wormhole-sdk/lib/esm/utils/consts";
import { config } from "dotenv";
import { homedir } from "os";

config({ path: `${homedir()}/.wormhole/.env` });

const getEnvVar = (varName: string): string | undefined => process.env[varName];

export type Connection = {
  rpc: string | undefined;
  key: string | undefined;
};

export type ChainConnections = {
  [chain in ChainName]: Connection;
};

const MAINNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "https://api.mainnet-beta.solana.com",
    key: getEnvVar("SOLANA_KEY"),
  },
};

const TESTNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "https://api.devnet.solana.com",
    key: getEnvVar("SOLANA_KEY_TESTNET"),
  },
};

const DEVNET = {
  unset: {
    rpc: undefined,
    key: undefined,
  },
  solana: {
    rpc: "http://localhost:8899",
    key: "J2D4pwDred8P9ioyPEZVLPht885AeYpifsFGUyuzVmiKQosAvmZP4EegaKFrSprBC5vVP1xTvu61vYDWsxBNsYx",
  },
};

/**
 *
 * If you get a type error here, it means that a chain you just added does not
 * have an entry in TESTNET.
 * This is implemented as an ad-hoc type assertion instead of a type annotation
 * on TESTNET so that e.g.
 *
 * ```typescript
 * TESTNET['solana'].rpc
 * ```
 * has type 'string' instead of 'string | undefined'.
 *
 * (Do not delete this declaration!)
 */
const isTestnetConnections: ChainConnections = TESTNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isMainnetConnections: ChainConnections = MAINNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isDevnetConnections: ChainConnections = DEVNET;

export const NETWORKS = { MAINNET, TESTNET, DEVNET };
