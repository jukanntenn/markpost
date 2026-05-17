import "@testing-library/jest-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import ThemeToggle from "./ThemeToggle";
import { ThemeProvider } from "../components/theme-provider";
import { useTheme } from "next-themes";
import { renderWithProviders, mockMatchMedia } from "../test/utils";

function renderWithTheme() {
  return renderWithProviders(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>
  );
}

function ThemeValueReader() {
  const { theme } = useTheme();
  return <span data-testid="theme-value">{theme}</span>;
}

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
  mockMatchMedia();
});

describe("ThemeToggle", () => {
  it("renders theme toggle button", () => {
    renderWithTheme();
    const toggle = screen.getByRole("button", { name: /toggle theme/i });
    expect(toggle).toBeInTheDocument();
  });

  it("shows system icon by default", () => {
    renderWithTheme();
    const toggle = screen.getByRole("button", { name: /toggle theme/i });
    const svg = toggle.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg).toHaveClass("lucide");
  });

  it("has menu trigger attributes", () => {
    renderWithTheme();
    const toggle = screen.getByRole("button", { name: /toggle theme/i });
    expect(toggle).toHaveAttribute("aria-haspopup", "menu");
  });

  it("renders within theme provider context", () => {
    renderWithProviders(
      <ThemeProvider>
        <ThemeToggle />
        <ThemeValueReader />
      </ThemeProvider>
    );
    const toggle = screen.getByRole("button", { name: /toggle theme/i });
    expect(toggle).toBeInTheDocument();
    expect(screen.getByTestId("theme-value")).toHaveTextContent("system");
  });
});
