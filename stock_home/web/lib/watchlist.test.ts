import { describe, expect, it } from "vitest";
import { addSymbols, createGroup, DEFAULT_GROUP_ID, deleteGroup, MAX_SYMBOLS, parseCsv, parseStore, readWatchlist, removeSymbols, toCsv } from "./watchlist";

describe("watchlist store", () => {
  it("migrates the legacy symbol array and drops malformed entries", () => {
    const s = parseStore(JSON.stringify(["RELIANCE", "bad!", "RELIANCE", 3]));
    expect(s.groups).toEqual([{ id: DEFAULT_GROUP_ID, name: "My Watchlist", symbols: ["RELIANCE"] }]);
    expect(parseStore("{not json").groups[0].symbols).toEqual([]);
    const g = parseStore(JSON.stringify({ groups: [{ id: "it", name: "  IT  ", symbols: ["TCS"] }, { id: 1 }] }));
    expect(g.groups.map((x) => x.id)).toEqual([DEFAULT_GROUP_ID, "it"]);
    expect(g.groups[1].name).toBe("IT");
    expect(readWatchlist({ getItem: () => JSON.stringify({ groups: [{ id: DEFAULT_GROUP_ID, name: "Mine", symbols: ["INFY"] }] }) })).toEqual(["INFY"]);
  });

  it("adds within the limit, removes, and manages groups", () => {
    let s = parseStore(null);
    const many = Array.from({ length: MAX_SYMBOLS + 3 }, (_, i) => `S${i}`);
    const res = addSymbols(s, DEFAULT_GROUP_ID, ["TCS", "TCS", ...many]);
    expect(res.added).toHaveLength(MAX_SYMBOLS);
    s = removeSymbols(res.store, DEFAULT_GROUP_ID, ["TCS"]);
    expect(s.groups[0].symbols).not.toContain("TCS");

    const created = createGroup(s, "Swing Trade");
    expect(created?.id).toBe("swing-trade");
    expect(createGroup(created!.store, "swing trade")).toBeNull();
    expect(createGroup(created!.store, "   ")).toBeNull();
    expect(deleteGroup(created!.store, DEFAULT_GROUP_ID).groups).toHaveLength(2);
    expect(deleteGroup(created!.store, "swing-trade").groups).toHaveLength(1);
  });

  it("reads and writes CSV", () => {
    expect(parseCsv('symbol,name\n"infy",Infosys\r\nTCS\n\nINFY\n')).toEqual(["INFY", "TCS"]);
    expect(toCsv({ id: "x", name: "X", symbols: ["A", "B"] })).toBe("symbol\nA\nB\n");
  });
});
