import {
  afterAll,
  beforeAll,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import axios from "axios";
import base58 from "bs58";
import { eth } from "web3";
import {
  PerChainQueryRequest,
  QueryProxyMock,
  QueryProxyQueryResponse,
  QueryRequest,
  QueryResponse,
  SolanaAccountQueryRequest,
  SolanaAccountQueryResponse,
  SolanaPdaEntry,
  SolanaPdaQueryRequest,
  SolanaPdaQueryResponse,
} from "..";

jest.setTimeout(120000);

const SOLANA_NODE_URL = "http://localhost:8899";
const QUERY_URL = "https://testnet.ccq.vaa.dev/v1/query";

const SOL_PDAS: SolanaPdaEntry[] = [
  {
    programAddress: Uint8Array.from(
      base58.decode("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
    ), // Core Bridge address
    seeds: [
      new Uint8Array(Buffer.from("GuardianSet")),
      new Uint8Array(Buffer.alloc(4)),
    ], // Use index zero in tilt.
  },
];

let mock: QueryProxyMock;

beforeAll(() => {
  mock = new QueryProxyMock({
    1: SOLANA_NODE_URL,
  });
});

afterAll(() => {});

describe.skip("mocks match testnet", () => {
  test("SolAccount to devnet", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest("finalized", accounts)
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a0901000000000000000000000000000000000000000000000000000000000000000000000000"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
    );
  });
  test("SolAccount to devnet with min context slot", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest("finalized", accounts, BigInt(7))
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a0901000000000000000000000000000000000000000000000000000000000000000000000000"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
    );
  });
  test("SolAccount to devnet with data slice", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest(
          "finalized",
          accounts,
          BigInt(0),
          BigInt(1),
          BigInt(10)
        )
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
  });
  test("SolAccount to devnet with min context slot and data slice", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest(
          "finalized",
          accounts,
          BigInt(7),
          BigInt(1),
          BigInt(10)
        )
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
  });
  test("SolanaPda to devnet", async () => {
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaPdaQueryRequest(
          "finalized",
          SOL_PDAS,
          BigInt(0),
          BigInt(12),
          BigInt(16) // After this, things can change.
        )
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0].response as SolanaPdaQueryResponse;
    expect(sar.blockTime).not.toEqual(BigInt(0));
    expect(sar.results.length).toEqual(1);

    expect(Buffer.from(sar.results[0].account).toString("hex")).toEqual(
      "4fa9188b339cfd573a0778c5deaeeee94d4bcfb12b345bf8e417e5119dae773e"
    );
    expect(sar.results[0].bump).toEqual(253);
    expect(sar.results[0].lamports).toEqual(BigInt(1141440));
    expect(sar.results[0].rentEpoch).toEqual(BigInt(0));
    expect(sar.results[0].executable).toEqual(false);
    expect(Buffer.from(sar.results[0].owner).toString("hex")).toEqual(
      "02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa"
    );
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "57cd18b7f8a4d91a2da9ab4af05d0fbe"
    );
  });
});
