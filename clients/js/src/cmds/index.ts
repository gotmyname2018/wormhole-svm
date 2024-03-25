// https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
import * as editVaa from "./editVaa";
import * as generate from "./generate";
import * as info from "./info";
import * as parse from "./parse";
import * as recover from "./recover"; //TBDel
import * as submit from "./submit";
import * as transfer from "./transfer";
import * as verifyVaa from "./verifyVaa"; //TBDel
import * as status from "./status";

// Commands can be imported as an array of commands.
// Documentation about command hierarchy can be found here: https://github.com/yargs/yargs/blob/main/docs/advanced.md#example-command-hierarchy-using-indexmjs
export const CLI_COMMAND_MODULES = [
  editVaa,
  generate,
  info,
  parse,
  recover,
  submit,
  transfer,
  verifyVaa,
  status,
];
