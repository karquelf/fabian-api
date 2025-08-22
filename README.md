# fabian-api

## Usage

```
go run .
```

```
curl -i -H "Content-Type: application/json" -d '{ "message": "What time is it?" }'  http://localhost:3000/text
```

```
curl -i -F "audio=@../fabian-stt/bonjour.wav" http://localhost:3000/voice
```
