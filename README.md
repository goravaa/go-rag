# Go-RAG
<img width="1024" height="499" alt="image" src="https://github.com/user-attachments/assets/5af110ed-31b8-429d-b234-c981db69518a" />

## Dev note
I'm developing Go-RAG, a modular knowledge retrieval system that solves the issue of personal knowledge management for complex subjects from maths to codebases. I am aiming for complete local deployment with minimal usage of resources. I am aiming for it to be enterprise grade system with multiple people using it, integrating asynch file system.

* Current State * : I have created authentication middleware, embedding service, and chunking strategies using AST trees but accuracy of the retrieval is not as much as i wanted for my topics to solve this i have stopped on this and currently developing fast GraphRAG engine in rust for low resource environment to achieve better retrieval after testing with graphrag python library. Context: https://arxiv.org/pdf/2404.16130

# System Design of Go Rag
<img width="903" height="545" alt="image" src="https://github.com/user-attachments/assets/e8a413ef-8188-49ba-9b02-64f1d09312f5" />
