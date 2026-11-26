# fresh-studio

a fun game studio

### a proposed architecture (thanks to Chat GPT)

Go	go1.27.4 windows/amd64
Git	2.49.0.windows.1
Node.js	v22.22.2
npm	10.9.7
Ollama	0.40.16

### Check a local Ollama model

Create a `.env` file in the repository root with an already installed model:

```dotenv
OLLAMA_MODEL=qwen3.5:2b
OLLAMA_HOST=http://localhost:11434
```

Run `ollama list` to see installed models, then run this command from the
repository root:

```powershell
go run .\services\orchestrator\cmd\modelcheck
```

The command loads `.env` from the current working directory. Existing process
environment variables take precedence, and can be used without a `.env` file.

```
ai-game-studio/
│
├── apps/
│   ├── dashboard/
│   └── game-preview/
│
├── services/
│   ├── api/
│   ├── orchestrator/
│   ├── agent-runtime/
│   ├── model-router/
│   └── evaluator/
│
├── packages/
│   ├── contracts/
│   ├── game-core/
│   ├── game-ui/
│   └── game-testing/
│
├── agents/
│   ├── creative-director/
│   ├── game-designer/
│   ├── engineer/
│   ├── art-director/
│   ├── qa/
│   └── reviewer/
│
├── templates/
│   ├── word-game/
│   ├── puzzle/
│   ├── arcade/
│   └── endless-runner/
│
├── games/
│
├── infrastructure/
│
└── docs/
```
