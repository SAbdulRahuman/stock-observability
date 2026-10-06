import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { Money, PrivacyProvider, usePrivacy } from "./privacy";

function Toggle() {
  const { toggle } = usePrivacy();
  return (
    <button type="button" onClick={toggle}>
      toggle
    </button>
  );
}

describe("Money", () => {
  beforeEach(() => window.localStorage.clear());

  it("hides amounts by default and reveals them on toggle", () => {
    render(
      <PrivacyProvider>
        <Money value={1240000} />
        <Toggle />
      </PrivacyProvider>,
    );
    expect(screen.getByLabelText("hidden amount")).toHaveTextContent("₹ ••••");
    fireEvent.click(screen.getByText("toggle"));
    expect(screen.getByText("₹12,40,000")).toBeInTheDocument();
    expect(window.localStorage.getItem("stock-home:hide-amounts")).toBe("false");
  });

  it("remembers the revealed state", () => {
    window.localStorage.setItem("stock-home:hide-amounts", "false");
    render(
      <PrivacyProvider>
        <Money value={-1150} signed />
      </PrivacyProvider>,
    );
    expect(screen.getByText("-₹1,150")).toBeInTheDocument();
  });
});
