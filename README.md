# fresh-studio

a fun game studio

### a proposed architecture (thanks to Chat GPT)

Go	go1.27.4 windows/amd64
Git	2.49.0.windows.1
Node.js	v22.22.2
npm	10.9.7
Ollama	0.40.16

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

### Generate a game concept

Start the orchestrator from the repository root, with the Ollama settings above
and Ollama running:

```powershell
go run .\services\orchestrator
```

In another PowerShell window, send a task:

```powershell
$task = @{
    id = "example-1"
    objective = "Design a casual puzzle game"
    context = "For mobile players"
    constraints = @("Short sessions")
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://localhost:8080/concepts -ContentType "application/json" -Body $task
```

The response contains the agent result and its concept in `output`. This
endpoint does not persist the task or result. The request runs for at most
three minutes, and only one concept can be generated at a time; concurrent
requests receive HTTP 429. The server listens on localhost only and has no
authentication. It requires `OLLAMA_MODEL` at startup; it does not download
models.
