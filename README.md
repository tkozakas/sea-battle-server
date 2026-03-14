# Sea Battle Server

Backend for [Sea Battle](https://github.com/tkozakas/sea-battle-web), an online multiplayer battleship game. Built with Go, WebSockets, and chi router.

## Run locally

```
cp env.template .env
go run cmd/server/main.go
```

Or with Docker:

```
cp env.template .env
docker compose up
```

Server starts on `http://localhost:8080` by default. See `env.template` for configuration options.

## Test

```
go test ./... -race
```

## License

[MIT](LICENSE)
