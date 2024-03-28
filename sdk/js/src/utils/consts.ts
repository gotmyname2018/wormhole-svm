export const CHAINS = {
  unset: 0,
  solana: 1,
} as const;

export type ChainName = keyof typeof CHAINS;
export type ChainId = typeof CHAINS[ChainName];

/*
 *
 * All the Solana-based chain names that Wormhole supports
 */
export const SolanaChainNames = ["solana", "pythnet"] as const;
export type SolanaChainName = typeof SolanaChainNames[number];

export type Contracts = {
  core: string | undefined;
  token_bridge: string | undefined;
  nft_bridge: string | undefined;
};

export type ChainContracts = {
  [chain in ChainName]: Contracts;
};

export type Network = "MAINNET" | "TESTNET" | "DEVNET";

const MAINNET = {
  unset: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  solana: {
    core: "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth",
    token_bridge: "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb",
    nft_bridge: "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD",
  },
};

const TESTNET = {
  unset: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  solana: {
    core: "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
    token_bridge: "DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe",
    nft_bridge: "2rHhojZ7hpu1zA91nvZmT8TqWWvMcKmmNBCr2mKTtMq4",
  },
};

const DEVNET = {
  unset: {
    core: undefined,
    token_bridge: undefined,
    nft_bridge: undefined,
  },
  solana: {
    core: "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o",
    token_bridge: "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE",
    nft_bridge: "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA",
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
 * TESTNET['solana'].core
 * ```
 * has type 'string' instead of 'string | undefined'.
 *
 * (Do not delete this declaration!)
 */
const isTestnetContracts: ChainContracts = TESTNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isMainnetContracts: ChainContracts = MAINNET;

/**
 *
 * See [[isTestnetContracts]]
 */
const isDevnetContracts: ChainContracts = DEVNET;

/**
 *
 * Contracts addresses on testnet and mainnet
 */
export const CONTRACTS = { MAINNET, TESTNET, DEVNET };

// We don't specify the types of the below consts to be [[ChainId]]. This way,
// the inferred type will be a singleton (or literal) type, which is more precise and allows
// typescript to perform context-sensitive narrowing when checking against them.
// See the [[isEVMChain]] for an example.
export const CHAIN_ID_UNSET = CHAINS["unset"];
export const CHAIN_ID_SOLANA = CHAINS["solana"];

// This inverts the [[CHAINS]] object so that we can look up a chain by id
export type ChainIdToName = {
  -readonly [key in keyof typeof CHAINS as typeof CHAINS[key]]: key;
};
export const CHAIN_ID_TO_NAME: ChainIdToName = Object.entries(CHAINS).reduce(
  (obj, [name, id]) => {
    obj[id] = name;
    return obj;
  },
  {} as any
) as ChainIdToName;

/**
 *
 * All the EVM-based chain ids that Wormhole supports
 */
export type EVMChainId = typeof CHAINS[SolanaChainName]; // TBDel

/**
 *
 * All the Solana-based chain ids that Wormhole supports
 */
export type SolanaChainId = typeof CHAINS[SolanaChainName];

/**
 *
 * Returns true when called with a valid chain, and narrows the type in the
 * "true" branch to [[ChainId]] or [[ChainName]] thanks to the type predicate in
 * the return type.
 *
 * A typical use-case might look like
 * ```typescript
 * foo = isChain(c) ? doSomethingWithChainId(c) : handleInvalidCase()
 * ```
 */
export function isChain(chain: number | string): chain is ChainId | ChainName {
  if (typeof chain === "number") {
    return chain in CHAIN_ID_TO_NAME;
  } else {
    return chain in CHAINS;
  }
}

/**
 *
 * Asserts that the given number or string is a valid chain, and throws otherwise.
 * After calling this function, the type of chain will be narrowed to
 * [[ChainId]] or [[ChainName]] thanks to the type assertion in the return type.
 *
 * A typical use-case might look like
 * ```typescript
 * // c has type 'string'
 * assertChain(c)
 * // c now has type 'ChainName'
 * ```
 */
export function assertChain(
  chain: number | string
): asserts chain is ChainId | ChainName {
  if (!isChain(chain)) {
    if (typeof chain === "number") {
      throw Error(`Unknown chain id: ${chain}`);
    } else {
      throw Error(`Unknown chain: ${chain}`);
    }
  }
}

export function toChainId(chainName: ChainName): ChainId {
  return CHAINS[chainName];
}

export function toChainName(chainId: ChainId): ChainName {
  return CHAIN_ID_TO_NAME[chainId];
}

export function coalesceChainId(chain: ChainId | ChainName): ChainId {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return typeof chain === "number" && isChain(chain) ? chain : toChainId(chain);
}

export function coalesceChainName(chain: ChainId | ChainName): ChainName {
  // this is written in a way that for invalid inputs (coming from vanilla
  // javascript or someone doing type casting) it will always return undefined.
  return toChainName(coalesceChainId(chain));
}

export function isSolanaChain(
  chain: ChainId | ChainName
): chain is SolanaChainId | SolanaChainName {
  const chainName = coalesceChainName(chain);
  return SolanaChainNames.includes(chainName as unknown as SolanaChainName);
}

export const WSOL_ADDRESS = "So11111111111111111111111111111111111111112";
export const WSOL_DECIMALS = 9;
export const MAX_VAA_DECIMALS = 8;
