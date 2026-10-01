import { describe, expect, it } from "vitest";
import { validWaldoCode, validWaldoLabel } from "./waldo-connection";

describe("Waldo boundary validation (synthetic inputs)", () => {
 it("accepts canonical unpadded 32-byte codes and rejects pad-bit aliases", () => {
  expect(validWaldoCode("A".repeat(43))).toBe(true);
  expect(validWaldoCode("_".repeat(42) + "8")).toBe(true);
  for (const value of ["A".repeat(42), "A".repeat(44), "A".repeat(42)+"B", "A".repeat(42)+"=", "A".repeat(42)+"/", "A".repeat(42)+"+", " A".repeat(22), "Ａ".repeat(43), "A".repeat(43)+"\n"]) expect(validWaldoCode(value)).toBe(false);
 });
 it("counts UTF-8 bytes without truncating or replacing malformed Unicode", () => {
  for (const value of ["x", "x".repeat(120), "é".repeat(60), "機".repeat(40), "😀".repeat(30)]) expect(validWaldoLabel(value)).toBe(true);
  for (const value of ["", "x".repeat(121), "é".repeat(61), "機".repeat(41), "😀".repeat(31), "\ud800", "\udc00", "x\ud800y"]) expect(validWaldoLabel(value)).toBe(false);
 });
});
