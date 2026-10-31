# Task Planning Prompt

Decompose the user's objective into structured tasks.
Specify:
- title: concise title
- description: clear requirements
- agent: "coding-agent" | "writing-agent" | "investment-agent" | "video-agent"
- dependencies: list of dependent task indices
- acceptance_criteria: verifiable completion checks
- risk: "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"
