"use client";

import { useCallback, useEffect, useState } from "react";

type Pipeline = { id: string; name: string; tasks: unknown[] };
type Run = {
  id: string;
  pipelineId: string;
  status: "pending" | "running" | "succeeded" | "failed" | "canceled";
  createdAt: string;
};

const api = "/api/control-plane";
const samplePipeline = {
  id: "hello-pipeline",
  name: "Hello pipeline",
  tasks: [
    { id: "hello", image: "local", command: ["echo", "hello from runner"] },
  ],
};

export default function Home() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [message, setMessage] = useState("Control Plane에 연결하는 중입니다.");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [pipelineResponse, runResponse] = await Promise.all([
        fetch(`${api}/pipelines`),
        fetch(`${api}/runs`),
      ]);
      if (!pipelineResponse.ok || !runResponse.ok)
        throw new Error("Control Plane 응답을 받을 수 없습니다.");
      setPipelines((await pipelineResponse.json()).pipelines);
      setRuns((await runResponse.json()).runs);
      setMessage("최신 상태입니다.");
    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : "알 수 없는 오류가 발생했습니다.",
      );
    }
  }, []);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  async function createSample() {
    setBusy(true);
    try {
      const response = await fetch(`${api}/pipelines`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(samplePipeline),
      });
      if (!response.ok)
        throw new Error(
          (await response.json()).message ??
            "파이프라인을 저장하지 못했습니다.",
        );
      await refresh();
      setMessage("샘플 파이프라인을 저장했습니다.");
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : "저장에 실패했습니다.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function startRun(pipelineId: string) {
    setBusy(true);
    try {
      const response = await fetch(`${api}/runs`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pipelineId }),
      });
      if (!response.ok)
        throw new Error(
          (await response.json()).message ?? "실행을 시작하지 못했습니다.",
        );
      await refresh();
      setMessage(
        "새 실행을 생성했습니다. Runner가 작업을 가져갈 때까지 기다립니다.",
      );
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : "실행 생성에 실패했습니다.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <main>
      <header>
        <div>
          <p className="eyebrow">DEVOPS PLATFORM · MVP</p>
          <h1>Pipeline Console</h1>
        </div>
        <button onClick={() => void refresh()} disabled={busy}>
          새로고침
        </button>
      </header>
      <p className="message" role="status">
        {message}
      </p>
      <section className="panel">
        <div className="section-heading">
          <div>
            <h2>파이프라인</h2>
            <p>정의를 저장하고 실행을 시작합니다.</p>
          </div>
          <button
            className="primary"
            onClick={() => void createSample()}
            disabled={busy}
          >
            샘플 만들기
          </button>
        </div>
        {pipelines.length === 0 ? (
          <div className="empty">
            아직 파이프라인이 없습니다. 샘플부터 만들어 보세요.
          </div>
        ) : (
          <div className="list">
            {pipelines.map((pipeline) => (
              <article key={pipeline.id}>
                <div>
                  <h3>{pipeline.name}</h3>
                  <p>
                    {pipeline.id} · 작업 {pipeline.tasks.length}개
                  </p>
                </div>
                <button
                  className="primary"
                  onClick={() => void startRun(pipeline.id)}
                  disabled={busy}
                >
                  실행
                </button>
              </article>
            ))}
          </div>
        )}
      </section>
      <section className="panel">
        <div className="section-heading">
          <div>
            <h2>최근 실행</h2>
            <p>3초마다 자동 갱신됩니다.</p>
          </div>
        </div>
        {runs.length === 0 ? (
          <div className="empty">실행 기록이 없습니다.</div>
        ) : (
          <div className="list">
            {runs.map((run) => (
              <article key={run.id}>
                <div>
                  <h3>{run.id}</h3>
                  <p>
                    {run.pipelineId} ·{" "}
                    {new Date(run.createdAt).toLocaleString("ko-KR")}
                  </p>
                </div>
                <span className={`status ${run.status}`}>{run.status}</span>
              </article>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}
