import { describe, expect, test } from "@jest/globals";
import {
  QueryResponse,
} from "..";

describe("from works with hex and Uint8Array", () => {
  test("QueryResponse", () => {
    const result =
      "010000b094a2ee9b1d5b310e1710bb5f6106bd481f28d932f83d6220c8ffdd5c55b91818ca78c812cd03c51338e384ab09265aa6fb2615a830e13c65775e769c5505800100000037010000002a010005010000002a0000000930783238343236626201130db1b83d205562461ed0720b37f1fbc21bf67f00000004916d5743010005010000005500000000028426bb7e422fe7df070cd5261d8e23280debfd1ac8c544dcd80837c5f1ebda47c06b7f000609c35ffdb8800100000020000000000000000000000000000000000000000000000000000000000000002a";
    const queryResponseFromHex = QueryResponse.from(result);
    const queryResponseFromUint8Array = QueryResponse.from(
      Buffer.from(result, "hex")
    );
    expect(queryResponseFromHex.serialize()).toEqual(
      queryResponseFromUint8Array.serialize()
    );
  });
});
