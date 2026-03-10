from fastapi import FastAPI
from .service import status

app = FastAPI()


@app.get("/py/health")
async def health():
    return {"ok": status()}
