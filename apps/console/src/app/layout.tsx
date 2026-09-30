import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  title: "DevOps Platform Console",
  description: "MVP pipeline and run console",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="ko">
      <body>{children}</body>
    </html>
  );
}
