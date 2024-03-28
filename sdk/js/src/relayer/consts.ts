import { ethers } from "ethers";
import { ChainName, Network } from "../";

type AddressInfo = {
  wormholeRelayerAddress?: string;
  mockDeliveryProviderAddress?: string;
  mockIntegrationAddress?: string;
};

const TESTNET: { [K in ChainName]?: AddressInfo } = {
};

const DEVNET: { [K in ChainName]?: AddressInfo } = {
};

const MAINNET: { [K in ChainName]?: AddressInfo } = {
};

export const RELAYER_CONTRACTS = { MAINNET, TESTNET, DEVNET };

export function getAddressInfo(
  chainName: ChainName,
  env: Network
): AddressInfo {
  const result: AddressInfo | undefined = RELAYER_CONTRACTS[env][chainName];
  if (!result) throw Error(`No address info for chain ${chainName} on ${env}`);
  return result;
}

export function getWormholeRelayerAddress(
  chainName: ChainName,
  env: Network
): string {
  const result = getAddressInfo(chainName, env).wormholeRelayerAddress;
  if (!result)
    throw Error(
      `No Wormhole Relayer Address for chain ${chainName}, network ${env}`
    );
  return result;
}

export function getWormholeRelayer(
  chainName: ChainName,
  env: Network,
  provider: ethers.providers.Provider | ethers.Signer,
  wormholeRelayerAddress?: string
): WormholeRelayer {
  const thisChainsRelayer =
    wormholeRelayerAddress || getWormholeRelayerAddress(chainName, env);
  const contract = WormholeRelayer__factory.connect(
    thisChainsRelayer,
    provider
  );
  return contract;
}

export const RPCS_BY_CHAIN: {
  [key in Network]: { [key in ChainName]?: string };
} = {
  MAINNET: {
  },
  TESTNET: {
  },
  DEVNET: {
  },
};

export const GUARDIAN_RPC_HOSTS = [
  "https://wormhole-v2-mainnet-api.certus.one",
  "https://wormhole.inotel.ro",
  "https://wormhole-v2-mainnet-api.mcf.rocks",
  "https://wormhole-v2-mainnet-api.chainlayer.network",
  "https://wormhole-v2-mainnet-api.staking.fund",
];

export const getWormscanAPI = (_network: Network) => {
  switch (_network) {
    case "MAINNET":
      return "https://api.wormholescan.io/";
    case "TESTNET":
      return "https://api.testnet.wormholescan.io/";
    default:
      // possible extension for tilt/ci - search through the guardian api
      // at localhost:7071 (tilt) or guardian:7071 (ci)
      throw new Error("Not testnet or mainnet - so no wormscan api access");
  }
};
