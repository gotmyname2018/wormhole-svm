import yargs from "yargs";
import { impossible } from "../vaa";
import { CHAIN_NAME_CHOICES, NETWORK_OPTIONS, NETWORKS } from "../consts";
import { assertNetwork } from "../utils";
import { transferSolana } from "../solana";

export const command = "transfer";
export const desc = "Transfer a token";
export const builder = (y: typeof yargs) =>
  y
    .option("src-chain", {
      describe: "source chain",
      choices: CHAIN_NAME_CHOICES,
      demandOption: true,
    })
    .option("dst-chain", {
      describe: "destination chain",
      choices: CHAIN_NAME_CHOICES,
      demandOption: true,
    })
    .option("dst-addr", {
      describe: "destination address",
      type: "string",
      demandOption: true,
    })
    .option("token-addr", {
      describe: "token address",
      type: "string",
      default: "native",
      defaultDescription: "native token",
      demandOption: false,
    })
    .option("amount", {
      describe: "token amount",
      type: "string",
      demandOption: true,
    })
    .option("network", NETWORK_OPTIONS)
    .option("rpc", {
      describe: "RPC endpoint",
      type: "string",
      demandOption: false,
    });

export const handler = async (
  argv: Awaited<ReturnType<typeof builder>["argv"]>
) => {
  const srcChain = argv["src-chain"];
  const dstChain = argv["dst-chain"];
  if (srcChain === "unset") {
    throw new Error("source chain is unset");
  }
  if (dstChain === "unset") {
    throw new Error("destination chain is unset");
  }
  if (srcChain === dstChain) {
    throw new Error("source and destination chains can't be the same");
  }
  const amount = argv.amount;
  if (BigInt(amount) <= 0) {
    throw new Error("amount must be greater than 0");
  }
  const tokenAddr = argv["token-addr"];
  const dstAddr = argv["dst-addr"];
  const network = argv.network.toUpperCase();
  assertNetwork(network);
  const rpc = argv.rpc ?? NETWORKS[network][srcChain].rpc;
  if (!rpc) {
    throw new Error(`No ${network} rpc defined for ${srcChain}`);
  }
  if (srcChain === "solana") {
    await transferSolana(
      srcChain,
      dstChain,
      dstAddr,
      tokenAddr,
      amount,
      network,
      rpc
    );
  } else {
    // If you get a type error here, hover over `chain`'s type and it tells you
    // which cases are not handled
    impossible(srcChain);
  }
};
