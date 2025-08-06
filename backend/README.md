# Full Stack Master Sync - Backend

## Environment Setup

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Update the `.env` file with your actual values:
   - `OPENAI_API_KEY`: Your OpenAI API key from https://platform.openai.com/api-keys
   - Other configuration values as needed

## Running the Application

```bash
go run main.go
```

## Testing

```bash
go test ./...
```

## WebSocket Endpoints

- `/ws` - WebSocket connection endpoint
- `/ws/stats` - WebSocket statistics endpoint

## API Endpoints

- `/health` - Health check endpoint
- `/api` - API base endpoint with information