# Project: RAG-based Support Bot

## 1. Overview

A Retrieval-Augmented Generation (RAG) system designed to answer user queries using company documentation.

## 2. Technology Stack

- **Languages:** Python 3.10
- **Frameworks:** LangChain, FastAPI
- **Database:** Pinecone (vector store)
- **LLM API:** OpenAI API (GPT-4o)
- **Authentication:** OAuth2 (Auth0)
- **Infrastructure:** Kubernetes with HPA

## 3. Entry Points

- `src/main.py` — FastAPI application bootstrap
- `src/cli.py` — CLI for document ingestion

## 4. Directory Structure

```
src/
  api/          — HTTP route handlers
  core/         — Query orchestration (LangChain)
  embeddings/   — Document chunking and embedding pipeline
  models/       — Data models and schemas
  config/       — Environment and app configuration
tests/          — Unit and integration tests
scripts/        — Deployment and ingestion scripts
```

## 5. Data Flow

1. User submits query via UI.
2. Query is converted to embedding.
3. Vector DB returns relevant context.
4. Prompt + Context sent to LLM.
5. Answer streamed back to user.

## 6. Design Patterns

- **Retrieval-Augmented Generation** — LLM responses grounded in retrieved documents
- **Chain of Responsibility** — LangChain pipeline stages (embed → retrieve → prompt → generate)
- **Repository Pattern** — Pinecone access abstracted behind a vector store interface

## 7. External Integrations

- **OpenAI API** — Embedding generation and LLM response generation
- **Pinecone** — Vector similarity search
- **Auth0** — User authentication and token validation

## 8. Build & Run

```bash
pip install -r requirements.txt    # Install dependencies
python src/main.py                 # Start the API server
pytest tests/                      # Run tests
python src/cli.py ingest docs/     # Ingest documents
```

## 9. Domain Specifications

- [Chat System](./chat-system/chat-system.md)
- [Embedding Pipeline](./embeddings/embedding-pipeline.md)
- [LLM Voting Mechanism](./llm-voting/llm-voting.md)

## 10. Domain Relationships

<!-- List inter-domain dependencies showing which domain depends on which. -->
<!-- Use → to indicate dependency direction: "A → B" means A depends on B. -->

- Chat System → Embedding Pipeline (queries use embeddings for retrieval)
- Chat System → LLM Voting Mechanism (responses scored by voting)
- Embedding Pipeline → (no internal dependencies)
