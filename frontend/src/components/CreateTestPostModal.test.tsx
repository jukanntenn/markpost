import "@testing-library/jest-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import CreateTestPostModal from "./CreateTestPostModal";
import { ThemeProvider } from "../contexts/ThemeProvider";
import { setMockAuth, createWrapper } from "../test/utils";
import { server } from "../mocks/server";
import { http, HttpResponse } from "msw";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

function renderWithProviders(ui: React.ReactElement) {
  const wrapper = createWrapper();
  return render(<ThemeProvider>{wrapper({ children: ui })}</ThemeProvider>);
}

const mockOnHide = vi.fn();
const mockOnSuccess = vi.fn();

beforeEach(() => {
  vi.clearAllMocks();
  setMockAuth({
    access_token: "test_token",
    refresh_token: "test_refresh",
    user: { id: 1, username: "testuser" },
  });

  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

describe("CreateTestPostModal", () => {
  it("renders modal when show is true", () => {
    renderWithProviders(
      <CreateTestPostModal
        show
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    expect(screen.getByRole("dialog")).toBeVisible();
  });

  it("does not render modal when show is false", () => {
    renderWithProviders(
      <CreateTestPostModal
        show={false}
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("disables submit button when body is empty", () => {
    renderWithProviders(
      <CreateTestPostModal
        show
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    const createButton = screen.getByRole("button", { name: /create/i });
    expect(createButton).toBeDisabled();
  });

  it("enables submit button when body is not empty", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <CreateTestPostModal
        show
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    const bodyTextarea = screen.getByPlaceholderText(/content/i);
    await user.type(bodyTextarea, "Test content");

    const createButton = screen.getByRole("button", { name: /create/i });
    expect(createButton).toBeEnabled();
  });

  it("submits form successfully", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <CreateTestPostModal
        show
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    const titleInput = screen.getByPlaceholderText(/title/i);
    const bodyTextarea = screen.getByPlaceholderText(/content/i);

    await user.type(titleInput, "Test Title");
    await user.type(bodyTextarea, "Test content");

    const createButton = screen.getByRole("button", { name: /create/i });
    await user.click(createButton);

    await waitFor(() => {
      expect(mockOnSuccess).toHaveBeenCalled();
    });
  });

  it("closes modal when cancel is clicked", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <CreateTestPostModal
        show
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    const cancelButton = screen.getByRole("button", { name: /cancel/i });
    await user.click(cancelButton);

    expect(mockOnHide).toHaveBeenCalled();
  });

  it("handles server error", async () => {
    server.use(
      http.post("/:postKey", () => {
        return HttpResponse.json({ error: "Server error" }, { status: 500 });
      })
    );

    const user = userEvent.setup();
    renderWithProviders(
      <CreateTestPostModal
        show
        postKey="test_key"
        onHide={mockOnHide}
        onSuccess={mockOnSuccess}
      />
    );

    const bodyTextarea = screen.getByPlaceholderText(/content/i);
    await user.type(bodyTextarea, "Test content");

    const createButton = screen.getByRole("button", { name: /create/i });
    await user.click(createButton);

    await waitFor(() => {
      expect(screen.getByText(/server error/i)).toBeInTheDocument();
    });
  });
});
