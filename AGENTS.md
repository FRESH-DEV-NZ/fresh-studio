
# AI Game Studio

## Project principles

- Build a local-first autonomous AI game development platform.
- Backend: Go.
- Frontend: TypeScript and React.
- Game engine: Phaser.
- Database: SQLite initially.
- Local model runtime: Ollama.

## Engineering standards

- Prefer simple, modular architecture.
- Use explicit interfaces between components.
- Write tests for important functionality.
- Never silently fall back to cloud inference.
- Never automatically download models.
- Keep agent execution bounded and auditable.
- Do not allow agents unrestricted filesystem or command access.
- Keep secrets out of source control.
- Document architectural decisions.

## Development environment

- Operating system: Windows 11.
- Shell: PowerShell.
- Go and Node.js are installed natively.
