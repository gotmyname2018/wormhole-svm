const ci = !!process.env.CI;

// see devnet.md
export const SOLANA_HOST = ci
  ? "http://solana-devnet:8899"
  : "http://localhost:8899";
export const SOLANA_PRIVATE_KEY = new Uint8Array([
  14, 173, 153, 4, 176, 224, 201, 111, 32, 237, 183, 185, 159, 247, 22, 161, 89,
  84, 215, 209, 212, 137, 10, 92, 157, 49, 29, 192, 101, 164, 152, 70, 87, 65,
  8, 174, 214, 157, 175, 126, 98, 90, 54, 24, 100, 177, 247, 77, 19, 112, 47,
  44, 165, 109, 233, 102, 14, 86, 109, 29, 134, 145, 132, 141,
]);
export const TEST_SOLANA_TOKEN = "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ";
export const WORMHOLE_RPC_HOSTS = ci
  ? ["http://guardian:7071"]
  : ["http://localhost:7071"];

export type Environment = "devnet" | "testnet" | "mainnet";
export const CLUSTER: Environment = "devnet" as Environment; //This is the currently selected environment.
