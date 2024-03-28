import { describe, expect, it } from "@jest/globals";
import { Connection, PublicKey } from "@solana/web3.js";

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
export const SOLANA_PRIVATE_KEY2 = new Uint8Array([
  118, 84, 4, 83, 83, 183, 31, 184, 20, 172, 95, 146, 7, 107, 141, 183, 124,
  196, 66, 246, 215, 243, 54, 61, 118, 188, 239, 237, 168, 108, 227, 169, 93,
  119, 180, 216, 9, 169, 30, 4, 167, 235, 188, 51, 70, 24, 181, 227, 189, 59,
  163, 161, 252, 219, 17, 105, 197, 241, 19, 66, 205, 188, 232, 131,
]);
export const TEST_SOLANA_TOKEN = "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna";
export const TEST_SOLANA_TOKEN3 =
  "AQJc65JzbzsT88JnGEXSqZaF8NFAXPo21fX4QUED4uRX";
export const WORMHOLE_RPC_HOSTS = ci
  ? ["http://guardian:7071"]
  : ["http://localhost:7071"];

describe("consts should exist", () => {
  it("has Solana test token", () => {
    expect.assertions(1);
    const connection = new Connection(SOLANA_HOST, "confirmed");
    return expect(
      connection.getAccountInfo(new PublicKey(TEST_SOLANA_TOKEN))
    ).resolves.toBeTruthy();
  });
});
