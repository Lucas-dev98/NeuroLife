from fastapi import FastAPI
from pydantic import BaseModel, Field


app = FastAPI(title="NeuroLife AI Service", version="0.1.0")


class DecomposeTaskRequest(BaseModel):
    title: str = Field(..., min_length=3)
    context: str | None = None


class DecomposeTaskResponse(BaseModel):
    title: str
    subtasks: list[str]


@app.get("/healthz")
def healthz() -> dict:
    return {"service": "ai-python", "status": "ok"}


@app.post("/v1/decompose-task", response_model=DecomposeTaskResponse)
def decompose_task(payload: DecomposeTaskRequest) -> DecomposeTaskResponse:
    # Mock inicial para Sprint 1. Substituir por pipeline de LLM na Sprint 7.
    base = payload.title.strip().rstrip(".")
    subtasks = [
        f"Definir objetivo para: {base}",
        "Quebrar em 3 a 5 etapas menores",
        "Executar primeira etapa em 15 minutos",
        "Revisar progresso e ajustar proximo passo",
    ]
    return DecomposeTaskResponse(title=base, subtasks=subtasks)
