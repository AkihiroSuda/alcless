import { expect, test, describe, mock } from "bun:test";
import { sudoers } from "./sudo.ts";

describe("Sudoers logic", () => {
  test("getContent should generate correct format", () => {
    const content = sudoers.getContent("host", "session");
    expect(content).toBe("host ALL=(root) NOPASSWD: /usr/bin/su - session *\n");
  });

  test("getFilePath should return /etc/sudoers.d/ path", () => {
    expect(sudoers.getFilePath("alcl_user_inst")).toBe("/etc/sudoers.d/alcl_user_inst");
  });
});
