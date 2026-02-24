# Specification: Chat System

## 1. Purpose

Handles real-time user queries by orchestrating the retrieval and generation pipeline. This is the primary user-facing domain — every query enters through the chat API and exits as a streamed LLM response.

## 2. Key Components

- `src/api/chat.py` — HTTP endpoint for submitting queries and streaming responses
- `src/core/orchestrator.py` — Coordinates the retrieve → prompt → generate pipeline
- `src/core/prompt_builder.py` — Constructs the LLM prompt from query + retrieved context
- `src/models/chat.py` — Data models: `ChatQuery`, `ChatResponse`, `ChatSession`

## 3. Data Models

- **ChatQuery** — `(session_id, user_id, query_text, timestamp)`
- **ChatResponse** — `(session_id, response_text, sources[], latency_ms)`
- **ChatSession** — `(id, user_id, created_at, messages[])`

## 4. Interfaces

- **API Endpoint**: `POST /api/chat` (Auth required) — Accepts a query, returns a streamed response
- **Internal**: `Orchestrator.run(query) -> ChatResponse` — Called by the chat endpoint
- **Internal**: `PromptBuilder.build(query, context[]) -> str` — Constructs the final LLM prompt

## 5. Dependencies

- **Depends on:** Embedding Pipeline (for vector retrieval), OpenAI API (for generation), Auth0 (for user validation)
- **Depended on by:** Frontend UI (via HTTP API)

## 6. Acceptance Criteria

- Authenticated user can submit a query and receive a streamed response.
- Response includes source references from retrieved documents.
- Unauthenticated requests return 401.
- Queries with no relevant context return a graceful fallback message.

## 7. Edge Cases

- Query text exceeds token limit — truncate and warn.
- Vector DB returns zero results — respond with "no relevant documents found" rather than hallucinating.
- LLM API timeout — return partial response if available, otherwise error.
