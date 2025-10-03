# Go-RAG
<img width="1024" height="499" alt="image" src="https://github.com/user-attachments/assets/5af110ed-31b8-429d-b234-c981db69518a" />

## Dev note
I'm developing Go-RAG, a modular knowledge retrieval system that solves the problem of fragmented and out-of-date information for technical teams. It syncs entire folders of Markdown and code into a vector database, creating a real-time, searchable knowledge base from a team's existing files. Unlike restrictive SaaS tools, Go-RAG is built for flexibility, enabling powerful use cases like a "chat with your repo" tool for faster developer onboarding, a fact-checking system for technical documentation, or a personal knowledge manager for conversational note retrieval. The system is built in Go with a robust microservice architecture (gRPC/REST APIs, Postgres, Qdrant, Google's Gemma embeddings) to deliver factually-grounded answers and significantly reduce developer search time.

# System Design of Go Rag
<img width="903" height="545" alt="image" src="https://github.com/user-attachments/assets/e8a413ef-8188-49ba-9b02-64f1d09312f5" />
