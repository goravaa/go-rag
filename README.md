# Go-RAG
<img width="1024" height="499" alt="image" src="https://github.com/user-attachments/assets/5af110ed-31b8-429d-b234-c981db69518a" />

## Dev note
I’m building Go-RAG — a modular knowledge retrieval system that fixes the mess of managing complex knowledge, from math concepts to large codebases. The goal is full local deployment with low resource usage, but still robust enough for enterprise use and multi-user async file handling.

* Currently: I’ve built the auth middleware, embedding service, and AST-based chunking. Retrieval accuracy isn’t where I want it, so I’ve paused that part to build a GraphRAG engine in Rust — faster, lighter, and built for better retrieval in constrained environments. Testing it against the Python GraphRAG implementation to push the system’s precision further.
[Context: https://arxiv.org/pdf/2404.16130]

# System Design of Go Rag
<img width="903" height="545" alt="image" src="https://github.com/user-attachments/assets/e8a413ef-8188-49ba-9b02-64f1d09312f5" />
