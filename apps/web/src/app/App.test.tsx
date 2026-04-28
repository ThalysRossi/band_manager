import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { App } from "./App";

describe("App", () => {
  it("renders the translated navigation", () => {
    render(<App />);

    expect(screen.getByRole("heading", { name: "Band Manager" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Inventory/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Merch Booth/i })).toBeInTheDocument();
  });
});
