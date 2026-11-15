import React, { CSSProperties } from "react";

export interface ShimmerSkeletonProps {
  width?: string | number;
  height?: string | number;
  borderRadius?: string | number;
  className?: string;
  style?: CSSProperties;
}

export const ShimmerSkeleton: React.FC<ShimmerSkeletonProps> = ({
  width = "100%",
  height = "16px",
  borderRadius = "4px",
  className = "",
  style = {},
}) => {
  return (
    <div
      className={`win-shimmer ${className}`}
      style={{
        width,
        height,
        borderRadius,
        ...style,
      }}
    />
  );
};

export const ServerCardSkeleton: React.FC = () => {
  return (
    <div className="server-card win-server-card-skeleton" style={{ padding: "20px", marginBottom: "16px" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "16px" }}>
        <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
          <ShimmerSkeleton width="38px" height="38px" borderRadius="50%" />
          <div>
            <ShimmerSkeleton width="180px" height="18px" style={{ marginBottom: "6px" }} />
            <ShimmerSkeleton width="110px" height="13px" />
          </div>
        </div>
        <div style={{ display: "flex", gap: "6px" }}>
          <ShimmerSkeleton width="90px" height="28px" borderRadius="4px" />
          <ShimmerSkeleton width="28px" height="28px" borderRadius="4px" />
        </div>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(130px, 1fr))", gap: "10px", marginBottom: "16px" }}>
        <ShimmerSkeleton width="100%" height="52px" borderRadius="6px" />
        <ShimmerSkeleton width="100%" height="52px" borderRadius="6px" />
        <ShimmerSkeleton width="100%" height="52px" borderRadius="6px" />
        <ShimmerSkeleton width="100%" height="52px" borderRadius="6px" />
      </div>
      <div>
        <ShimmerSkeleton width="100%" height="86px" borderRadius="6px" />
      </div>
    </div>
  );
};
