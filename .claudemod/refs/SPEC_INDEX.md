# Project: RAG-based Support Bot

## 1. Overview

A Retrieval-Augmented Generation (RAG) system designed to answer user queries using company documentation.

## 2. System Components

- **User Interface:** Frontend web app.
- **Orchestrator:** Python service (LangChain) managing query flow.
- **Vector Database:** Pinecone storing embedded document chunks.
- **LLM:** OpenAI GPT-4o for response generation.

## 3. Data Flow

1. User submits query via UI.
2. Query is converted to embedding.
3. Vector DB returns relevant context.
4. Prompt + Context sent to LLM.
5. Answer streamed back to user.

## 4. Technology Stack

- **Languages:** Python 3.10
- **Frameworks:** LangChain, FastAPI
- **Database:** Pinecone
- **LLM API:** OpenAI API

## 5. Security & Scaling

- **Authentication:** OAuth2 (Auth0)
- **Scaling:** Kubernetes HPA (Horizontal Pod Autoscaler)

## 6. Feature Specifications

- [./chat-system](Chat System)
- [./llm-voting](LLM Voting Mechanism)
