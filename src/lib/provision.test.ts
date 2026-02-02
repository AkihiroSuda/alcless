import { expect, test, describe } from "bun:test";
import { provisionSession } from "./provision.ts";

describe("Provisioning logic", () => {
  test("provisionSession should be defined", () => {
    expect(provisionSession).toBeDefined();
  });
});
