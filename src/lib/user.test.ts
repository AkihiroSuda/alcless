import { expect, test, describe, spyOn } from "bun:test";
import { getSessionUsername, getHostUser } from "./user.ts";
import * as exec from "./exec.ts";

describe("User logic", () => {
  test("getSessionUsername should follow the pattern", async () => {
    // We can't easily mock whoami without mocking the run function
    // But we know it should start with alcl_ and contain the host user and instance name
    const username = await getSessionUsername("test");
    const hostUser = await getHostUser();
    expect(username).toBe(`alcl_${hostUser}_test`);
  });
});
