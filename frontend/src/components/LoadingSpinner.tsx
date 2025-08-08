import React from "react";
import { Spinner } from "react-bootstrap";

interface LoadingSpinnerProps {
  size?: "sm" | undefined;
  text?: string;
  className?: string;
}

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = undefined,
  text = "Loading...",
  className = "",
}) => {
  return (
    <div
      className={`d-flex flex-column align-items-center justify-content-center ${className}`}
    >
      <Spinner
        animation="border"
        role="status"
        size={size}
        className="mb-2 text-primary"
        style={{
          width: size === "sm" ? "1.25rem" : "1.5rem",
          height: size === "sm" ? "1.25rem" : "1.5rem",
        }}
      >
        <span className="visually-hidden">{text}</span>
      </Spinner>
      {text && <span className="text-muted small">{text}</span>}
    </div>
  );
};

export default LoadingSpinner;
